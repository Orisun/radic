// Date         : 2018/7/28
// Author       : zhangchaoyang
// Description  :
package core

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"math"
	"os"
	"path"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	storage "github.com/Orisun/radic/forward_index"
	"github.com/Orisun/radic/types"
	"github.com/Orisun/radic/util"
	"github.com/Orisun/radic/util/inmem"

	"log"

	"github.com/cespare/xxhash"
)

const (
	BATCH_DELETE_PERSENT = 1e-3 //待删文章占总文档数的BATCH_DELETE_PERSENT时才真正从倒排链上删除
	INVERTED_SEG_LEN     = 5e4  //多协程并行遍历倒排链，一个负责遍历INVERTED_SEG_LEN个元素
)

var (
	OutOfFuture time.Time
)

func init() {
	OutOfFuture = time.Now().Add(time.Hour * 10000)
}

type Indexer struct {
	invertedTables     []map[string][]uint64 //倒排索引是一组map，每个map在读写时会对整个map加锁，分成多个map可以降低锁冲突
	invertedTableLocks []sync.RWMutex        //倒排索引的每一个map上加一把读写锁
	forwardTable       inmem.Cache           //正排索引，只缓存热数据，全量数据存在dbs中
	dbs                []storage.Storage
	totalDocs          uint64 //一共索引了多少文档
	initOptions        types.IndexerOpts
	initialized        bool //是否已调用过Init函数

	addChannel    chan types.DocInfo //从此channel取出数据，从正排索引和倒排索引上将对应的doc删掉
	deleteChannel chan uint32        //从此channel取出数据，从正排索引上将对应的doc删掉
	// shutDown           chan bool          //通知add和delete子协和退出
	stoped             bool //置为true之后，所有请求都拒绝执行
	addRoutineFinished sync.WaitGroup
	delRoutineFinished sync.WaitGroup
	toBeDeleteBuffer   *util.ConcurrentMapString //从倒排索引上检出数据后，如果发现在这个集合里，则从倒排索引上删掉。sync.Map是go1.9中新加入的线程安全的map，性能没有ConcurrentMap高
	docScores          *util.ConcurrentMapUint32

	taskCount     sync.WaitGroup
	invertListLen *util.ConcurrentMapString //大概记录每个倒排链的长度，串行从多个倒排链取交集时先取短的

	haveStatistics     bool
	execDeleteThresh   int       //要删除的doc累积到一定量execDeleteThresh时，才执行真正的删除操作
	traverseResultPool sync.Pool //每次遍历倒排链时都创建一个临时数组，以容纳遍历的结果，造成GC时间较长。所以采用对象池进行优化
}

//Close 关闭local db，释放倒排索引所占的内存
func (indexer *Indexer) Close() {
	if !indexer.initialized {
		log.Printf("indexer have not initialized")
		return
	}

	indexer.stoped = true
	// indexer.shutDown <- true
	runtime.Gosched()
	indexer.taskCount.Wait() //等正在执行的工作结束
	log.Printf("all task of indexer stoped")

	//关闭add和delete管道，分别等两组子协程结束
	close(indexer.addChannel)
	close(indexer.deleteChannel)
	indexer.addRoutineFinished.Wait()
	log.Printf("add doc routines finished")
	indexer.delRoutineFinished.Wait()
	log.Printf("delete doc routines finished")

	indexer.invertedTables = nil
	indexer.forwardTable = nil
	indexer.toBeDeleteBuffer = nil //从map中delete key根本不会释放内存，只能设为nil，静待GC
	indexer.docScores = nil
	runtime.GC()

	for _, db := range indexer.dbs {
		if err := db.Close(); err != nil {
			log.Fatalf("close local db failed: %v", err)
		}
	}
	indexer.initialized = false
	log.Printf("close all local db")
}

//Init 初始化参数
func (indexer *Indexer) Init(options types.IndexerOpts) {
	if indexer.initialized {
		return
	}

	options.Init()

	indexer.initOptions = options
	indexer.invertedTables = make([]map[string][]uint64, options.InvertIndexSegmentNum)
	indexer.invertedTableLocks = make([]sync.RWMutex, options.InvertIndexSegmentNum)
	for i := 0; i < options.InvertIndexSegmentNum; i++ {
		indexer.invertedTables[i] = make(map[string][]uint64, 10000) //每个invertedTable上预计有1万个key
		indexer.invertedTableLocks[i] = sync.RWMutex{}
	}
	indexer.forwardTable = inmem.NewLocked(options.ForwardIndexSize)
	indexer.dbs = make([]storage.Storage, options.DbShardNum*2)
	for i := 0; i < options.DbShardNum*2; i++ {
		if stg, err := storage.OpenStorage(options.DbType, path.Join(options.DbPath,
			options.IndexName+"."+strconv.Itoa(i))); err == nil {
			if stg == nil {
				panic("open storage failed")
			} else {
				indexer.dbs[i] = stg
			}
		} else {
			panic("open storage failed")
		}
	}
	indexer.initialized = true

	indexer.addChannel = make(chan types.DocInfo, options.AddCacheSize)
	indexer.deleteChannel = make(chan uint32, options.DeleteCacheSize)
	indexer.toBeDeleteBuffer = util.NewConcurrentMapString(8)
	if indexer.initOptions.PreRankCriteria != nil { //如果没有指定PreRankCriteria，则不需要用到indexer.docScores缓存
		indexer.docScores = util.NewConcurrentMapUint32(32)
	}
	// indexer.shutDown = make(chan bool, 2*(options.AddDocThreads+options.DeleteDocThreads))
	indexer.addRoutineFinished = sync.WaitGroup{}
	indexer.addRoutineFinished.Add(options.AddDocThreads)
	indexer.delRoutineFinished = sync.WaitGroup{}
	indexer.delRoutineFinished.Add(options.DeleteDocThreads)

	indexer.taskCount = sync.WaitGroup{}
	indexer.invertListLen = util.NewConcurrentMapString(64)
	indexer.execDeleteThresh = 1000
	indexer.traverseResultPool = sync.Pool{
		New: func() interface{} {
			brr := make([]uint32, indexer.initOptions.MaxInvertListLen) //brr的cap和len都是indexer.initOptions.MaxInvertListLen
			return brr
		},
	}

	//启动addDoc的消费线程
	for i := 0; i < options.AddDocThreads; i++ {
		go func() {
			for {
				if doc, ok := <-indexer.addChannel; ok {
					indexer.addDoc(doc)
				} else { //indexer.addChannel管道已关闭，且
					break
				}

			}
			// defer indexer.addRoutineFinished.Done()
			// stop := false
			// for !stop {
			// 	select {
			// 	case doc := <-indexer.addChannel: //如果addChannel已空，该行代码会一直阻塞，所以放在select里通过shutDown这个管道来强制它退出
			// 		indexer.addDoc(doc)
			// 	case <-indexer.shutDown:
			// 		indexer.shutDown <- true
			// 		stop = true
			// 	}
			// }
		}()
	}

	//启动deleteDoc的消费线程
	for i := 0; i < options.DeleteDocThreads; i++ {
		go func() {
			for {
				if docId, ok := <-indexer.deleteChannel; ok {
					indexer.deleteDoc(docId)
				} else {
					break
				}
			}
			// defer indexer.delRoutineFinished.Done()
			// stop := false
			// for !stop {
			// 	select {
			// 	case docId := <-indexer.deleteChannel: //如果deleteChannel已空，该行代码会一直阻塞，所以放在select里通过shutDown这个管道来强制它退出
			// 		indexer.deleteDoc(docId)
			// 	case <-indexer.shutDown:
			// 		indexer.shutDown <- true
			// 		stop = true
			// 	}
			// }
		}()
	}
	indexer.stoped = false
}

func (indexer *Indexer) GetDbPath() string {
	return indexer.initOptions.DbPath
}

func (indexer *Indexer) getShardId(docId uint32) (uint64, uint64) {
	hasher := xxhash.Sum64String(strconv.FormatUint(uint64(docId), 10))
	shardId := hasher % uint64(indexer.initOptions.DbShardNum)
	return shardId, shardId + uint64(indexer.initOptions.DbShardNum)
}

func (indexer *Indexer) getIndexId(key string) uint64 {
	hasher := xxhash.Sum64String(key)
	indexId := hasher % uint64(indexer.initOptions.InvertIndexSegmentNum)
	return indexId
}

//getStorageKey
func (indexer *Indexer) getStorageKey(docId uint32) []byte {
	key := make([]byte, 8)
	cnt := binary.PutUvarint(key, uint64(docId))
	return key[:cnt]
}

//storageKey2DocId 与getStorageKey互为逆操作
func (indexer *Indexer) storageKey2DocId(key []byte) uint32 {
	num, _ := binary.Uvarint(key)
	return uint32(num)
}

//IterIndex 遍历索引里的数据
func (indexer *Indexer) IterIndex(fn func(docId uint32, docInfo types.DocInfo) error) {
	if indexer.stoped {
		log.Printf("index engine is stopped")
		return
	}
	indexer.taskCount.Add(1)
	defer indexer.taskCount.Done()
	for _, db := range indexer.dbs {
		buf := bytes.NewReader([]byte{})
		db.IterDB(func(k, v []byte) error {
			buf.Reset(v)
			dec := gob.NewDecoder(buf)
			var data types.DocInfo
			err := dec.Decode(&data)
			if err == nil {
				fn(data.DocId, data)
			}
			return nil
		})
	}
}

//IterKeyOnIndex 遍历索引里的key
func (indexer *Indexer) IterKeyOnIndex(fn func(docId uint32) error) error {
	if indexer.stoped {
		log.Printf("index engine is stopped")
		return nil
	}
	indexer.taskCount.Add(1)
	defer indexer.taskCount.Done()
	var rect error
	for _, db := range indexer.dbs {
		db.IterKey(func(k []byte) error {
			fn(indexer.storageKey2DocId(k))
			return nil
		})
	}
	return rect
}

func CombineDocidAndFlag(docid uint32, compositeFeature uint32) uint64 {
	return (uint64(docid) << 32) | uint64(compositeFeature)
}

func DisassembleDocidAndFlag(c uint64) (uint32, uint32) {
	docid := uint32(c >> 32)
	compositeFeature := uint32(c << 32 >> 32)
	return docid, compositeFeature
}

func (indexer *Indexer) addDoc(doc types.DocInfo) {
	indexer.toBeDeleteBuffer.Remove(strconv.FormatUint(uint64(doc.DocId), 10))
	indexer.addDocInner(doc, true)
}

func (indexer *Indexer) addDocInner(doc types.DocInfo, Insert2Storage bool) {
	if !indexer.initialized {
		log.Printf("indexer have not initialized")
		return
	}
	if indexer.docScores != nil {
		indexer.docScores.Set(doc.DocId, doc.RankScore)
	}
	//往倒排索引上添加
	indexer.AddToInvertIndex(doc.Keyword, &doc)

	if Insert2Storage {
		//往正排索引上添加
		indexer.AddToLocalDB(&doc)
	}
}

func (indexer *Indexer) AddToInvertIndex(keywords []*types.Keyword, doc *types.DocInfo) {
	if !indexer.initialized {
		log.Printf("indexer have not initialized")
		return
	}
	for _, keyword := range keywords {
		k := keyword.ToString()
		indexId := indexer.getIndexId(k)
		invertedTable := indexer.invertedTables[indexId]
		indexer.invertedTableLocks[indexId].Lock()
		if list, exists := invertedTable[k]; exists {
			invertedTable[k] = append(list, CombineDocidAndFlag(doc.DocId, doc.CompositeFeature)) //如果倒排链上已有该docid，则会出现重复
		} else {
			invertedTable[k] = []uint64{CombineDocidAndFlag(doc.DocId, doc.CompositeFeature)}
		}
		indexer.invertedTableLocks[indexId].Unlock()
	}
}

func (indexer *Indexer) AddToLocalDB(doc *types.DocInfo) {
	//往正排索引上添加
	keys := doc.Keyword
	doc.Keyword = nil //keys和detail分开存储，因为它们的读取频率不一样
	var value bytes.Buffer
	encoder := gob.NewEncoder(&value)           //gob是go自带的序列化方法
	if err := encoder.Encode(doc); err == nil { //由于DocInfo中的Entry是interface{}，没有指定具体的类型，所以在调用AddDoc之前需要先往gob中注册Entry的具体类型
		shardId1, shardId2 := indexer.getShardId(doc.DocId)
		detailDB := indexer.dbs[shardId1]
		storageKey := indexer.getStorageKey(doc.DocId)
		detailDB.Set(storageKey, value.Bytes())
		// fmt.Printf("add doc %d to shard %d %d\n", doc.DocId, shardId1, shardId2)

		var value2 bytes.Buffer
		encoder2 := gob.NewEncoder(&value2)
		if err := encoder2.Encode(keys); err == nil {
			keysDB := indexer.dbs[shardId2]
			keysDB.Set(storageKey, value2.Bytes())
		}
		atomic.AddUint64(&indexer.totalDocs, 1)
	} else {
		log.Printf("write doc %d to db failed", doc.DocId)
	}
}

//DocNum 索引中的文档总数。若indexer.initOptions.PreRankCriteria为nil，则返回-1
func (indexer *Indexer) DocNum() int {
	if indexer.stoped {
		log.Printf("index engine is stopped")
		return 0
	}
	indexer.taskCount.Add(1)
	defer indexer.taskCount.Done()
	if indexer.docScores != nil {
		t := indexer.docScores.Count()
		indexer.execDeleteThresh = int(float64(t) * BATCH_DELETE_PERSENT)
		return t
	} else {
		return -1
	}
}

func (indexer *Indexer) BatchIndexDoc(originDocs []*types.DocInfo) {
	if indexer.stoped {
		log.Printf("index engine is stopped")
		return
	}
	indexer.taskCount.Add(1)
	defer indexer.taskCount.Done()
	if !indexer.initialized {
		log.Printf("indexer have not initialized")
		return
	}

	//先深拷贝一份
	docs := append(make([]*types.DocInfo, 0, len(originDocs)), originDocs...)
	//往倒排索引上添加
	queue := make(chan *types.DocInfo, len(docs))
	for _, doc := range docs {
		if indexer.docScores != nil {
			indexer.docScores.Set(doc.DocId, doc.RankScore)
		}
		queue <- doc
		//util.Log.Debugf("add %d to queue, keyword len %d", doc.DocId,len(doc.Keyword))
	}
	close(queue)
	//并行写倒排索引
	writeInvertWg := sync.WaitGroup{}
	writeInvertWg.Add(indexer.initOptions.AddDocThreads)
	for i := 0; i < indexer.initOptions.AddDocThreads; i++ {
		go func() {
			defer writeInvertWg.Done()
			for doc := range queue { //多线程并行去消费这个queue
				keywords := doc.Keyword
				//util.Log.Debugf("key word len is %d of doc %d", len(keywords), doc.DocId)
				for _, keyword := range keywords {
					k := keyword.ToString()
					indexId := indexer.getIndexId(k)
					invertedTable := indexer.invertedTables[indexId]
					indexer.invertedTableLocks[indexId].Lock()
					if list, exists := invertedTable[k]; exists {
						invertedTable[k] = append(list, CombineDocidAndFlag(doc.DocId, doc.CompositeFeature)) //如果倒排链上已有该docid，则会出现重复
					} else {
						invertedTable[k] = []uint64{CombineDocidAndFlag(doc.DocId, doc.CompositeFeature)}
					}
					//util.Log.Debugf("add %d to invert list of key %s", doc.DocId, k)
					indexer.invertedTableLocks[indexId].Unlock()
				}
			}
		}()
	}
	writeInvertWg.Wait() //等待写倒排索引的任务结束

	//往正排索引上添加
	keysMap := make(map[uint64][][]byte, indexer.initOptions.DbShardNum)
	valuesMap := make(map[uint64][][]byte, indexer.initOptions.DbShardNum)

	for _, doc := range docs {
		detailShardId, keysShardId := indexer.getShardId(doc.DocId)
		keys := doc.Keyword
		doc.Keyword = nil
		storageKey := indexer.getStorageKey(doc.DocId)

		var detailWriter bytes.Buffer
		detailEncoder := gob.NewEncoder(&detailWriter)
		if err := detailEncoder.Encode(doc); err == nil {
			if keys, exists := keysMap[detailShardId]; !exists {
				keys = make([][]byte, 0, len(docs))
				values := make([][]byte, 0, len(docs))
				keys = append(keys, storageKey)
				values = append(values, detailWriter.Bytes())
				keysMap[detailShardId] = keys
				valuesMap[detailShardId] = values
			} else {
				keys = append(keys, storageKey)
				values := valuesMap[detailShardId]
				values = append(values, detailWriter.Bytes())
				keysMap[detailShardId] = keys
				valuesMap[detailShardId] = values
			}
		} else {
			log.Printf("write doc %d to db failed", doc.DocId)
		}

		var keysWriter bytes.Buffer
		keysEncoder := gob.NewEncoder(&keysWriter)
		if err := keysEncoder.Encode(keys); err == nil {
			if keys, exists := keysMap[keysShardId]; !exists {
				keys = make([][]byte, 0, len(docs))
				values := make([][]byte, 0, len(docs))
				keys = append(keys, storageKey)
				values = append(values, keysWriter.Bytes())
				keysMap[keysShardId] = keys
				valuesMap[keysShardId] = values
			} else {
				keys = append(keys, storageKey)
				values := valuesMap[keysShardId]
				values = append(values, keysWriter.Bytes())
				keysMap[keysShardId] = keys
				valuesMap[keysShardId] = values
			}
		} else {
			log.Printf("write doc %d to db failed", doc.DocId)
		}
	}

	//并行写本地db
	wg := sync.WaitGroup{}
	wg.Add(len(keysMap))
	for shardId := range keysMap {
		go func(shardId uint64) { //注意：在子协程中使用for range生成的变量时一定作为参数传给子协程
			defer wg.Done()
			db := indexer.dbs[shardId]
			keys := keysMap[shardId]
			values := valuesMap[shardId]
			db.BatchSet(keys, values)
			atomic.AddUint64(&indexer.totalDocs, uint64(len(keys)))
		}(shardId)
	}
	wg.Wait() //等待写local db的任务结束
}

func (indexer *Indexer) deleteDoc(docId uint32) {
	if !indexer.initialized {
		log.Printf("indexer have not initialized")
		return
	}
	//从docScores里删除
	if indexer.docScores != nil {
		indexer.docScores.Remove(docId)
	}

	//从正排索引上删除
	detailShardId, keysShardId := indexer.getShardId(docId)
	storageKey := indexer.getStorageKey(docId)
	detailDB := indexer.dbs[detailShardId]
	detailDB.Delete(storageKey)
	keysDB := indexer.dbs[keysShardId]

	//从正排索引上拿到doc有keywords后再【异步地】从倒排索引上删除
	keysReader := bytes.NewReader([]byte{})
	if keyBytes, err2 := keysDB.Get(storageKey); err2 == nil {
		if len(keyBytes) > 0 {
			keysReader.Reset(keyBytes)
			keysDecoder := gob.NewDecoder(keysReader)
			var keyDoc []*types.Keyword
			err3 := keysDecoder.Decode(&keyDoc)
			if err3 == nil {
				for _, ele := range keyDoc {
					//把要删除的docId存到deleteTable上。要删除的doc积累到一定量时才会从倒排上删除，所有经常会出现从倒排检出的结果在docScores和正排上没有，只要正排上没有，LookUp检索时就不会返回
					indexer.AddDeletePair(docId, ele)
				}
			}
		}
	}

	keysDB.Delete(storageKey)

}

//func (indexer *Indexer) haveBeenDeleted(docId uint32) bool {
//	return indexer.toBeDeleteBuffer.Has(strconv.FormatUint(uint64(docId), 10))
//}

//AddDeletePair 从倒排索引的一个key上把指定docID删掉。这是异步执行的，积累到一定量才会真正执行
func (indexer *Indexer) AddDeletePair(docId uint32, word *types.Keyword) {
	if indexer.stoped {
		log.Printf("index engine is stopped")
		return
	}
	indexer.taskCount.Add(1)
	defer indexer.taskCount.Done()
	key := strconv.FormatUint(uint64(docId), 10)
	if value, exists := indexer.toBeDeleteBuffer.Get(key); exists {
		conmap := value.(*util.ConcurrentMapString)
		conmap.Set(word.ToString(), true)
	} else {
		conmap := util.NewConcurrentMapString(4)
		conmap.Set(word.ToString(), true)
		indexer.toBeDeleteBuffer.Set(key, conmap)
	}
	if indexer.toBeDeleteBuffer.Count() > indexer.execDeleteThresh { //要删除的doc累积到一定量execDeleteThresh时，才执行真正的删除操作
		indexer.CleanToBeDeleteBuffer()
	}
}

//IndexDoc 往索引上添加一个doc。该操作中异步执行的
func (indexer *Indexer) IndexDoc(doc types.DocInfo) {
	if indexer.stoped {
		log.Printf("index engine is stopped")
		return
	}
	indexer.taskCount.Add(1)
	defer indexer.taskCount.Done()
	indexer.addChannel <- doc
}

//RemoveDoc 从索引上删除doc。该操作中异步执行的
func (indexer *Indexer) RemoveDoc(docId uint32) {
	if indexer.stoped {
		log.Printf("index engine is stopped")
		return
	}
	indexer.taskCount.Add(1)
	defer indexer.taskCount.Done()
	indexer.deleteChannel <- docId
}

//Lookup 根据条件去索引上搜索doc
func (indexer *Indexer) Lookup(must []*types.Keyword, should []*types.Keyword, not []*types.Keyword, orderLess bool, filterIds map[uint32]bool, onFlag uint32, offFlag uint32, orFlags []uint32) []*types.DocInfo {
	if indexer.stoped {
		log.Printf("index engine is stopped")
		return nil
	}
	indexer.taskCount.Add(1)
	defer indexer.taskCount.Done()
	//util.Log.Debugf("must", must, "should", should)
	t0 := time.Now()
	haveMust := false //筛选条件里有must
	if len(must) > 0 {
		haveMust = true
	}
	haveshould := false //筛选条件里有should
	if len(should) > 0 {
		haveshould = true
	}
	if !haveMust && !haveshould { //筛选条件里既没must又没should
		//util.Log.Debugf("not haveMust and not haveshould")
		return nil
	}

	var mustResult map[uint32]bool
	var shouldResult map[uint32]bool
	var notResult map[uint32]bool

	wg := sync.WaitGroup{}
	wg.Add(3)
	go func() {
		defer wg.Done()
		mustResult = indexer.mustFind(must, filterIds, onFlag, offFlag, orFlags) //must：所有关键词都要命中
	}()
	go func() {
		defer wg.Done()
		notResult = indexer.shouldFind(not, filterIds, onFlag, offFlag, orFlags) //not：只需要命中一个关键词
	}()
	go func() {
		defer wg.Done()
		shouldResult = indexer.shouldFind(should, filterIds, onFlag, offFlag, orFlags) //should：只需要命中一个关键词
	}()
	wg.Wait()

	haveMustResult := false //根据must条件有搜索结果
	if len(mustResult) > 0 {
		haveMustResult = true
	}
	if haveMust && !haveMustResult {
		//util.Log.Debugf("have not must result")
		return nil
	}
	haveshouldResult := false //根据shpuld条件有搜索结果
	if len(shouldResult) > 0 {
		haveshouldResult = true
	}
	if haveshould && !haveshouldResult {
		//util.Log.Debugf("have not should result")
		return nil
	}

	//t1 := time.Now()
	//util.Log.Debugf("retrieve", t1.Sub(t0).Nanoseconds())

	var intersect map[uint32]bool

	if haveMust && haveshould {
		intersect = make(map[uint32]bool, len(mustResult))
		for k := range mustResult {
			if shouldResult[k] {
				intersect[k] = true
			}
		}
	} else if haveMust {
		intersect = mustResult
	} else if haveshould {
		intersect = shouldResult
	}

	//t2 := time.Now()
	//util.Log.Debugf("intersect", t2.Sub(t1).Nanoseconds())

	if len(intersect) > 0 {
		filterDocIds := make(map[uint32]bool, len(notResult)) //如果notResult是nil，则len(notResult)返回0

		if len(notResult) > 0 {
			for k := range notResult {
				filterDocIds[k] = true
				//util.Log.Debugf("add %d to filterDocIds", k)
			}
		}

		idList := make([]uint32, 0, len(intersect))
		for docId := range intersect {
			//util.Log.Debugf("deal %d", docId)
			if !filterDocIds[docId] {
				idList = append(idList, docId)
			}
		}

		//t3 := time.Now()
		//util.Log.Debugf("filter", t3.Sub(t2).Nanoseconds())

		//量太大，到search层精排序容易超时，强制截断，同时也减小了网络开销
		if len(idList) > indexer.initOptions.MaxRetrieve {
			if indexer.docScores != nil && indexer.initOptions.PreRankCriteria != nil {
				//截断前要先按preScore排序，保证截取的是得分最高的那一部分
				ranks := make(types.ScoredDocs, 0, len(intersect))
				for _, docId := range idList {
					if score, e2 := indexer.docScores.Get(docId); e2 {
						rs := types.ScoredDoc{
							DocId:     docId,
							RankScore: indexer.initOptions.PreRankCriteria.Score(score.(float32)),
						}
						ranks = append(ranks, rs)
					}
					// else {
					// log.Printf("can not find scores of doc %d", docId)
					// }
				}
				ranks.TopK(indexer.initOptions.MaxRetrieve) //只需要取TopK就可以了，没必要排序

				minLen := indexer.initOptions.MaxRetrieve
				if len(ranks) < minLen {
					minLen = len(ranks)
				}
				idList = idList[0:minLen] //截断，只取前MaxRetrieve个。不需要申请额外的内存空间
				for i := 0; i < minLen; i++ {
					idList[i] = ranks[i].DocId
				}
				//util.Log.Debugf("original have %d results, truncate to size %d", len(ranks), len(idList))
			} else {
				idList = idList[0:indexer.initOptions.MaxRetrieve] //截断，只取前MaxRetrieve个。不需要申请额外的内存空间
			}
		}
		//从正排索引上取数据
		docInfos := indexer.GetDocs(idList)

		ut := time.Since(t0).Nanoseconds() / 1e6
		if ut > 300 { //超过300ms打warn日志
			mustLog := make([]string, 0, len(must))
			for _, m := range must {
				mustLog = append(mustLog, m.ToString())
			}
			shouldLog := make([]string, 0, len(should))
			for _, m := range should {
				shouldLog = append(shouldLog, m.ToString())
			}
			log.Printf("find for must %s should %s use time %d milliseconds", mustLog, shouldLog, ut)
		}
		return docInfos
	} else {
		return nil
	}
}

//GetInvertListLen 获取倒排链的长度
func (indexer *Indexer) GetInvertListLen(keyword *types.Keyword, onFlag uint32, offFlag uint32, orFlags []uint32) int32 {
	key := keyword.ToString()
	indexId := indexer.getIndexId(key)
	invertedTable := indexer.invertedTables[indexId]
	indexer.invertedTableLocks[indexId].RLock()
	defer indexer.invertedTableLocks[indexId].RUnlock()
	arr, exists := invertedTable[key]
	if exists && len(arr) > 0 {
		dup := make(map[uint32]bool, len(arr)) //docID排重
		var count int32 = 0
		for _, doc := range arr {
			docId, flag := DisassembleDocidAndFlag(doc)
			if !dup[docId] {
				if (flag&onFlag == onFlag) && (flag&offFlag == 0) {
					orOk := true
					for _, orFlag := range orFlags {
						if orFlag > 0 && flag&orFlag <= 0 {
							orOk = false
						}
					}
					if orOk {
						count++
					}
				}
				dup[docId] = true
			}
		}
		return count
	} else {
		return 0
	}
}

//mustFind 关键词必须都命中
func (indexer *Indexer) mustFind(words []*types.Keyword, filterIds map[uint32]bool, onFlag uint32, offFlag uint32, orFlags []uint32) map[uint32]bool {
	if len(words) == 0 {
		return nil
	}

	begin := time.Now()
	/**
	多协程并发读slice或map都是OK的。
	多协程并发读写slice不会报错，但是如果不加锁可能会导致数据不一致；多协程并发读写map程序会报错fatal error: concurrent map writes。
	为什么slice允许并发读写，而map不允许呢？因为slice中的每个元素都有确定的地址（头指针加偏移量），而map中某个key下的value其地址是不固定的。
	有固定地址的称为variable，多协程并发读写是允许的。array、slice、struct中的每个元素（或成员变量）都可以被独立地addressed，都是variable。
	*/
	results := make([][]uint32, len(words))
	wg := sync.WaitGroup{}
	wg.Add(len(words))
	shortestListIndex := -1
	shortestListLen := math.MaxInt32 //取最短的那个长度
	for i, word := range words {
		go func(word *types.Keyword, i int) { //注意：在子协程中使用for range生成的变量时一定作为参数传给子协程
			defer wg.Done()
			key := word.ToString()
			indexId := indexer.getIndexId(key)
			invertedTable := indexer.invertedTables[indexId]
			indexer.invertedTableLocks[indexId].RLock()
			defer indexer.invertedTableLocks[indexId].RUnlock()
			arr, exists := invertedTable[key]
			if exists && len(arr) > 0 {
				results[i] = indexer.traverseInvertList(arr, onFlag, offFlag, orFlags)
				if len(results[i]) < shortestListLen {
					shortestListIndex = i
					shortestListLen = len(results[i]) //由于并发的缘故，shortestListLen不一定是最短的长度，但这个没关系，shortestListLen只是用来决定hitCountMap的Capacity
				}
			}
		}(word, i) //range产生的参数一定要按值传到routine里面去，因为range每轮迭代使用的是同一个地址
	}
	wg.Wait()

	if shortestListLen == math.MaxInt32 {
		return map[uint32]bool{}
	}
	//先把最短的那条放到hitCountMap里
	hitCountMap := make(map[uint32]int, shortestListLen)
	arr := results[shortestListIndex]
	for _, ele := range arr {
		if ele == 0 { //遍历到arr最后一个元素了
			break
		}
		hitCountMap[ele] = 1
	}
	indexer.traverseResultPool.Put(arr) //读完后归还给traverseResultPool

	//再遍历其余的，增加hitCountMap中的计数
	for i, arr := range results {
		if arr == nil || i == shortestListIndex {
			continue
		}
		if len(hitCountMap) > 0 {
			dup := make(map[uint32]bool, len(hitCountMap)) //对单条倒排上的元素进行排重
			for _, ele := range arr {
				if ele == 0 { //遍历到arr最后一个元素了
					break
				}
				if cnt, exists := hitCountMap[ele]; exists && !dup[ele] {
					hitCountMap[ele] = cnt + 1
					dup[ele] = true
				}
			}
		}
		indexer.traverseResultPool.Put(arr) //读完后归还给traverseResultPool
	}

	rect := make(map[uint32]bool, len(hitCountMap))
	for docId, cnt := range hitCountMap {
		if cnt >= len(words) {
			if !filterIds[docId] { //把filterIds中的过滤掉
				rect[docId] = true
			}
		}
	}

	ut := time.Since(begin).Nanoseconds() / 1e6
	if ut > 100 { //超过100ms打warn日志
		mustLog := make([]string, 0, len(words))
		for _, m := range words {
			mustLog = append(mustLog, m.ToString())
		}
		log.Printf("must find for %s use time %d milliseconds", mustLog, ut)
	}
	return rect
}

//traverseInvertList 遍历单条倒排链，取下满足onFlag、offFlag和orFlags的元素。返回的数组用完后注意归还给traverseResultPool
func (indexer *Indexer) traverseInvertList(arr []uint64, onFlag uint32, offFlag uint32, orFlags []uint32) []uint32 {
	rect := indexer.traverseResultPool.Get().([]uint32)
	segCount := (len(arr) + INVERTED_SEG_LEN - 1) / INVERTED_SEG_LEN
	wg := sync.WaitGroup{}
	wg.Add(segCount)
	var idx int32 = 0
	for segIndex := 0; segIndex < segCount; segIndex++ {
		go func(segIndex int) {
			defer wg.Done()
			begin := segIndex * INVERTED_SEG_LEN
			end := (segIndex + 1) * INVERTED_SEG_LEN
			for i := begin; i < len(arr) && i < end; i++ {
				docIdAndFlag := arr[i]
				docId, flag := DisassembleDocidAndFlag(docIdAndFlag)
				if docId > 0 && (flag&onFlag == onFlag) && (flag&offFlag == 0) { //确保有效元素都大于0
					orOk := true
					for _, orFlag := range orFlags {
						if orFlag > 0 && flag&orFlag <= 0 {
							orOk = false
						}
					}
					if orOk {
						index := atomic.AddInt32(&idx, 1) - 1
						if int(index) >= cap(rect) {
							break
						}
						(rect)[index] = docId
					}
				}
			}
		}(segIndex)
	}
	wg.Wait()
	if int(idx) < cap(rect) {
		(rect)[idx] = 0 //最后一个元素置为0
	}
	return rect //rect用完后注意归还给traverseResultPool
}

//shouldFind 命中一个关键词即可
func (indexer *Indexer) shouldFind(words []*types.Keyword, filterIds map[uint32]bool, onFlag uint32, offFlag uint32, orFlags []uint32) map[uint32]bool {
	if len(words) == 0 {
		return nil
	}

	begin := time.Now()
	results := make([][]uint32, len(words))
	wg := sync.WaitGroup{}
	wg.Add(len(words))
	var total int32 = 0
	for i, word := range words {
		go func(word *types.Keyword, i int) { //注意：在子协程中使用for range生成的变量时一定作为参数传给子协程
			defer wg.Done()
			key := word.ToString()
			indexId := indexer.getIndexId(key)
			invertedTable := indexer.invertedTables[indexId]
			indexer.invertedTableLocks[indexId].RLock()
			defer indexer.invertedTableLocks[indexId].RUnlock()
			arr, exists := invertedTable[key]
			if exists && len(arr) > 0 {
				results[i] = indexer.traverseInvertList(arr, onFlag, offFlag, orFlags)
				atomic.AddInt32(&total, int32(len(results[i])))
			}
		}(word, i) //range产生的参数一定要按值传到routine里面去，因为range每轮迭代使用的是同一个地址
	}
	wg.Wait()

	rect := make(map[uint32]bool, atomic.LoadInt32(&total))
	for _, arr := range results {
		if arr == nil {
			continue
		}
		for _, ele := range arr {
			if ele == 0 { //遍历到arr最后一个元素了
				break
			}
			if !filterIds[ele] { //把filterIds中的过滤掉
				rect[ele] = true
			}
		}
		indexer.traverseResultPool.Put(arr) //读完后归还给traverseResultPool
	}

	ut := time.Since(begin).Nanoseconds() / 1e6
	if ut > 100 { //超过100ms打warn日志
		mustLog := make([]string, 0, len(words))
		for _, m := range words {
			mustLog = append(mustLog, m.ToString())
		}
		log.Printf("should find for %s use time %d milliseconds", mustLog, ut)
	}

	return rect
}

//CleanToBeDeleteBuffer 把toBeDeleteBuffer里需要删的doc从倒排索引上删除，同时也从toBeDeleteBuffer里删除
func (indexer *Indexer) CleanToBeDeleteBuffer() {
	if indexer.stoped {
		log.Printf("index engine is stopped")
		return
	}
	indexer.taskCount.Add(1)
	defer indexer.taskCount.Done()
	docIdList := make([]uint32, 0, indexer.toBeDeleteBuffer.Count())
	keyWordDocMap := make(map[*types.Keyword]map[uint32]bool, 50*cap(docIdList))
	indexer.toBeDeleteBuffer.IterCb(func(key string, value interface{}) {
		if n, err := strconv.ParseInt(key, 10, 64); err == nil {
			docId := uint32(n)
			docIdList = append(docIdList, docId)
			keywordMap := value.(*util.ConcurrentMapString)
			keywordMap.IterCb(func(key string, v interface{}) {
				word := types.NewKeyword(key)
				if word == nil || len(word.Word) == 0 {
					return
				}
				if docMap, exists := keyWordDocMap[word]; exists {
					docMap[docId] = true
				} else {
					docMap := make(map[uint32]bool, 40)
					docMap[docId] = true
					keyWordDocMap[word] = docMap
				}
			})
		}

	})

	wg := sync.WaitGroup{}
	wg.Add(len(keyWordDocMap))
	for word, docMap := range keyWordDocMap {
		key := word.ToString()
		go func(key string, docMap map[uint32]bool) { //注意：在子协程中使用for range生成的变量时一定作为参数传给子协程
			defer wg.Done()
			//util.Log.Debugf("%d doc indexed by keyword %s %v\n",len(docMap),key,docMap)
			indexId := indexer.getIndexId(key)
			invertedTable := indexer.invertedTables[indexId]
			indexer.invertedTableLocks[indexId].RLock()
			arr, exists := invertedTable[key]
			if exists && len(arr) > 0 {
				remain := []uint64{} //删掉该删的，还剩下的doc list
				for _, docIdAndFlag := range arr {
					docId, _ := DisassembleDocidAndFlag(docIdAndFlag)
					if _, ok := docMap[docId]; !ok {
						remain = append(remain, docIdAndFlag)
					}
				}
				indexer.invertedTableLocks[indexId].RUnlock()
				indexer.invertedTableLocks[indexId].Lock()
				invertedTable[key] = remain
				indexer.invertedTableLocks[indexId].Unlock()
			} else {
				indexer.invertedTableLocks[indexId].RUnlock()
			}
		}(key, docMap)
	}
	wg.Wait()

	for _, docId := range docIdList {
		key := strconv.FormatUint(uint64(docId), 10)
		indexer.toBeDeleteBuffer.Remove(key)
	}
}

//GetDoc 根据ID获取一个doc的详情。flag=1仅获取detail，flag=0都获取
func (indexer *Indexer) GetDoc(docId uint32, flag int) *types.DocInfo {
	if indexer.stoped {
		log.Printf("index engine is stopped")
		return nil
	}
	indexer.taskCount.Add(1)
	defer indexer.taskCount.Done()
	if flag == 1 {
		//优先从缓存读
		if doc, exists := indexer.forwardTable.Get(docId); exists {
			value := doc.(*types.DocInfo)
			return value
		}
	}
	//再从db读
	var data types.DocInfo
	detailShardId, keysShardId := indexer.getShardId(docId)
	storageKey := indexer.getStorageKey(docId)
	if datasBytes, error := indexer.dbs[detailShardId].Get(storageKey); error == nil {
		if len(datasBytes) == 0 { //正排索引上没有这个doc，则需要从倒排索引上把它删掉
			//把要删除的docId存到deleteTable上
			indexer.AddDeletePair(docId, &types.Keyword{})
		} else {
			// fmt.Println("datasBytes", datasBytes)
			buf := bytes.NewReader(datasBytes)
			dec := gob.NewDecoder(buf)
			err := dec.Decode(&data)
			if err == nil {
				indexer.forwardTable.Add(docId, &data, OutOfFuture) //放入缓存，缓存里面没有keys
			}
		}
	} else {
		log.Printf("read local db faile: %v", error)
	}
	if flag == 0 && data.DocId > 0 {
		var keys []*types.Keyword
		if datasBytes, error := indexer.dbs[keysShardId].Get(storageKey); error == nil {
			buf := bytes.NewReader(datasBytes)
			dec := gob.NewDecoder(buf)
			err := dec.Decode(&keys)
			if err == nil {
				data.Keyword = keys
			}
		}
	}
	return &data
}

//GetDocs 根据ID获取一批doc的详情
func (indexer *Indexer) GetDocs(docIds []uint32) []*types.DocInfo {
	if indexer.stoped {
		log.Printf("index engine is stopped")
		return []*types.DocInfo{}
	}
	indexer.taskCount.Add(1)
	defer indexer.taskCount.Done()
	rect := make([]*types.DocInfo, 0, len(docIds))
	remain := make(map[uint64][]uint32, len(docIds)) //shardID-->[不在缓存中的docId]
	hitCacheCount := 0
	notHitCacheCount := 0
	//优先读缓存
	for _, docId := range docIds {
		if doc, exists := indexer.forwardTable.Get(docId); exists {
			value := doc.(*types.DocInfo)
			rect = append(rect, value)
			hitCacheCount++
		} else {
			notHitCacheCount++
			shardID, _ := indexer.getShardId(docId)
			if ll, exists := remain[shardID]; exists {
				ll = append(ll, docId)
				remain[shardID] = ll
			} else {
				remain[shardID] = []uint32{docId}
			}
		}
	}

	if len(remain) > 0 {
		//从各个db中并行读取
		queue := make(chan *types.DocInfo, len(docIds)-len(rect))
		wg := sync.WaitGroup{}
		wg.Add(len(remain))
		for shardId, docIdList := range remain {
			go func(shardId uint64, docIdList []uint32) { //注意：在子协程中使用for range生成的变量时一定作为参数传给子协程
				defer wg.Done()
				keys := make([][]byte, len(docIdList))
				for i, docId := range docIdList {
					keys[i] = indexer.getStorageKey(docId)
				}
				datasBytesArr, _ := indexer.dbs[shardId].BatchGet(keys) //BatchGet不保证顺序，即datasBytesArr和keys的顺序不一样
				byteReader := bytes.NewReader([]byte{})                 //reader重用
				for _, datasBytes := range datasBytesArr {
					byteReader.Reset(datasBytes)
					decoder := gob.NewDecoder(byteReader)
					var data types.DocInfo
					err := decoder.Decode(&data) //占实时搜索22%的耗时
					if err == nil {
						docId := data.DocId
						if docId > 0 {
							indexer.forwardTable.Add(docId, &data, OutOfFuture) //放入缓存
							queue <- &data
						}
					}
				}
			}(shardId, docIdList) //range产生的参数一定要按值传到routine里面去，因为range每轮迭代使用的是同一个地址
		}
		wg.Wait()

		close(queue) //把channel关闭后就可以无阻塞地遍历channel了

		for ele := range queue {
			rect = append(rect, ele)
		}
	}

	return rect
}

func (indexer *Indexer) LoadDataFromDb2InvertIndex() int {
	if indexer.stoped {
		log.Printf("index engine is stopped")
		return 0
	}
	indexer.taskCount.Add(1)
	defer indexer.taskCount.Done()
	var total int32 = 0
	for i := 0; i < indexer.initOptions.DbShardNum; i++ {
		detailDB := indexer.dbs[i]
		keysDB := indexer.dbs[i+indexer.initOptions.DbShardNum]
		detailReader := bytes.NewReader([]byte{})
		keysReader := bytes.NewReader([]byte{}) //reader重用
		detailDB.IterDB(func(k, v []byte) error {
			detailReader.Reset(v)
			detailDecoder := gob.NewDecoder(detailReader)
			var data types.DocInfo
			err := detailDecoder.Decode(&data)
			if err == nil {
				storageKey := indexer.getStorageKey(data.DocId)
				// fmt.Printf("doc id %d\n", data.DocId)
				if keyBytes, err2 := keysDB.Get(storageKey); err2 == nil {
					if len(keyBytes) > 0 {
						keysReader.Reset(keyBytes)
						keysDecoder := gob.NewDecoder(keysReader)
						var keyDoc []*types.Keyword
						err3 := keysDecoder.Decode(&keyDoc)
						if err3 == nil {
							data.Keyword = keyDoc
							indexer.addDocInner(data, false)
							atomic.AddInt32(&total, 1)
							if atomic.LoadInt32(&total)%100000 == 0 {
								log.Printf("load %d docs from local db to inverted index", atomic.LoadInt32(&total))
							}
						} else {
							log.Printf("decode key doc of %d failed", data.DocId)
						}
					} else {
						log.Printf("keys of %d not in local db", data.DocId)
					}
				} else {
					log.Printf("keys of %d not in local db", data.DocId)
				}
			} else {
				log.Printf("decode detail doc failed")
			}
			return err
		})
	}

	//indexer.duplicateInvertList()

	indexer.InvertIndexStatistics("")

	t := int(atomic.LoadInt32(&total))
	log.Printf("total load %d docs from local db to inverted index", t)
	indexer.execDeleteThresh = int(float64(t) * BATCH_DELETE_PERSENT)
	return t
}

func (indexer *Indexer) QuickLoadDataFromDb2InvertIndex() int {
	if indexer.stoped {
		log.Printf("index engine is stopped")
		return 0
	}
	indexer.taskCount.Add(1)
	defer indexer.taskCount.Done()
	var total int32
	wg := sync.WaitGroup{}
	wg.Add(len(indexer.dbs) / 2)
	for i := 0; i < indexer.initOptions.DbShardNum; i++ {
		go func(i int) { //并行加载db会非常耗内存
			defer wg.Done()
			detailDB := indexer.dbs[i]
			keysDB := indexer.dbs[i+indexer.initOptions.DbShardNum]
			cnt := 0
			detailReader := bytes.NewReader([]byte{})
			keyReader := bytes.NewReader([]byte{})
			detailDB.IterDB(func(k, v []byte) error {
				cnt++
				detailReader.Reset(v)
				detailDecoder := gob.NewDecoder(detailReader)
				var data types.DocInfo
				err := detailDecoder.Decode(&data)
				if err == nil {
					storageKey := indexer.getStorageKey(data.DocId)
					if keyBytes, err2 := keysDB.Get(storageKey); err2 == nil {
						if len(keyBytes) > 0 {
							keyReader.Reset(keyBytes)
							keysDecoder := gob.NewDecoder(keyReader)
							var keyDoc []*types.Keyword
							err3 := keysDecoder.Decode(&keyDoc)
							if err3 == nil {
								data.Keyword = keyDoc
								indexer.addDocInner(data, false)
								atomic.AddInt32(&total, 1)
								if atomic.LoadInt32(&total)%100000 == 0 {
									log.Printf("load %d docs from local db to inverted index", atomic.LoadInt32(&total))
								}
							} else {
								log.Printf("decode key doc of %d failed", data.DocId)
							}
						} else {
							log.Printf("keys of %d not in local db", data.DocId)
						}
					} else {
						log.Printf("keys of %d not in local db", data.DocId)
					}
				} else {
					log.Printf("decode detail doc failed")
				}
				return err
			})
			log.Printf("%d docs in %d db", cnt, i)
		}(i)
	}
	wg.Wait()

	//indexer.duplicateInvertList()

	indexer.InvertIndexStatistics("")

	t := int(atomic.LoadInt32(&total))
	log.Printf("total load %d docs from local db to inverted index", t)
	indexer.execDeleteThresh = int(float64(t) * BATCH_DELETE_PERSENT)
	return t
}

//InvertIndexStatistics 统计倒排索引上每个关键词下面有几个docID，给indexer.invertListLen赋值，返回倒排链的总数
func (indexer *Indexer) InvertIndexStatistics(outfile string) int {
	if indexer.stoped {
		log.Printf("index engine is stopped")
		return 0
	}
	indexer.taskCount.Add(1)
	defer indexer.taskCount.Done()
	KeywordCount := 0
	LongListCount := 0
	for idx, table := range indexer.invertedTables {
		indexer.invertedTableLocks[idx].RLock()
		KeywordCount += len(table)
		for kw, arr := range table {
			indexer.invertListLen.Set(kw, len(arr))
			if len(arr) > indexer.initOptions.MaxInvertListLen {
				LongListCount++
				log.Printf("key %s inverted list len %d", kw, len(arr))
			}

		}
		indexer.invertedTableLocks[idx].RUnlock()
	}
	indexer.haveStatistics = true

	log.Printf("一共%d条倒排链, 其中%d条长度超过了上限%d", KeywordCount, LongListCount, indexer.initOptions.MaxInvertListLen)

	if outfile != "" {
		os.Remove(outfile)
		f_out, err := os.OpenFile(outfile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			panic(err)
		}
		defer f_out.Close()
		buf_out := bufio.NewWriter(f_out)
		for k, v := range indexer.invertListLen.Items() {
			buf_out.WriteString(fmt.Sprintf("%s\t%d\n", k, v.(int)))
		}
		buf_out.Flush()
	}
	return KeywordCount
}

//duplicateInvertList 每条倒排链上存在重复的docid，这里进行去重
//func (indexer *Indexer) duplicateInvertList() {
//	util.Log.Infof("begin duplicate invert list")
//	begin := time.Now()
//	wg := sync.WaitGroup{}
//	wg.Add(len(indexer.invertedTables))
//	for idx, table := range indexer.invertedTables {
//		go func(idx int, table map[string][]uint32) {
//			defer wg.Done()
//			indexer.invertedTableLocks[idx].Lock()
//			for kw, arr := range table { //一边遍历map，一边修改value
//				dup := make(map[uint32]bool)
//				for _, ele := range arr {
//					dup[ele] = true
//				}
//				newList := make([]uint32, len(dup))
//				for k, _ := range dup {
//					newList = append(newList, k)
//				}
//				table[kw] = newList
//			}
//			indexer.invertedTableLocks[idx].Unlock()
//		}(idx, table)
//	}
//	wg.Wait()
//	util.Log.Infof("duplicate invert list use time {} seconds", time.Since(begin).Seconds())
//}
