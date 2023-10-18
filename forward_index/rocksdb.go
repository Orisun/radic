// package storage

// import (
// 	"log"
// 	"os"
// 	"path"
// 	"sync"
// 	"sync/atomic"
// 	"time"

// 	"github.com/line/gorocksdb"
// )

// //一些基本概念：
// //RocksDB大量复用了levedb的代码，并且还借鉴了许多HBase的设计理念。原始代码从leveldb 1.5 上fork出来。
// //SST文件（数据文件/SST表）:SST是Sorted Sequence Table（排序队列表）。他们是排好序的数据文件。在这些文件里，所有键都按爪排序好的顺序组织，一个键或者一个迭代位置可以通过二分查找进行定位。
// //前置写日志(WAL)或者日志(LOG)：一个在RocksDB重启的时候，用于恢复没有刷入SST文件的数据的文件。
// //memtable/写缓冲(write buffer)：在内存中存储最新更新的数据的数据结构。
// //Compression：压缩信息会存入磁盘，所以即使是用算法A进行的压缩，也可以用算法B打开文件。
// //Compaction：将一些SST文件合并成另外一些SST文件的后台任务。分层压缩或者基于分层的压缩方式：RocksDB的默认压缩方式，优先查询最新写入的level（即小level），以期望提高读性能。
// //Bloom filters：一个key可能会存在于多个SST文件中（因为update会转换为append操作），访问每个SST文件之前会先询问Bloom filter，key是否在里面。还有一种选择是为每个block建立一个Bloom filter，而不是为每个SST file建立一个Bloom filter。

// var (
// 	rocksOptions = gorocksdb.NewDefaultOptions()
// 	readOptions  = gorocksdb.NewDefaultReadOptions()
// 	writeOptions = gorocksdb.NewDefaultWriteOptions()
// )

// type Rocksdb struct {
// 	db              *gorocksdb.DB
// 	path            string
// 	timeWindowBegin time.Time
// 	load            int32
// }

// func OpenRocksdb(dbPath string) (Storage, error) {
// 	if err := os.MkdirAll(path.Dir(dbPath), os.ModePerm); err != nil { //如果dbPath对应的文件夹已存在则什么都不做，如果dbPath对应的文件已存在则返回错误
// 		return nil, err
// 	}

// 	rocksOptions.SetCreateIfMissing(true) //DB目录不存在时则创建之
// 	//memtable选用HashSkipList
// 	rocksOptions.SetAllowConcurrentMemtableWrites(false) //只有当memtable是skiplist时才支持并发write
// 	rocksOptions.SetPrefixExtractor(gorocksdb.NewFixedPrefixTransform(0))
// 	rocksOptions.SetHashSkipListRep(1000000, 4, 4) //需要指定PrefixExtractor

// 	////BlockBasedTable
// 	//blockBasedTableOptions := gorocksdb.NewDefaultBlockBasedTableOptions()
// 	////block_size: By default, this value is set to be 4k. If compression is enabled, a smaller block size would lead to higher random read speed because decompression overhead is reduced. But the block size cannot be too small to make compression useless. It is recommended to set it to be 1k.
// 	//blockBasedTableOptions.SetBlockSize(1024)
// 	////hash_index: In the new version, hash index is enabled for block based table. It would use 5% more storage space but speed up the random read by 50% compared to normal binary search index.
// 	//blockBasedTableOptions.SetIndexType(gorocksdb.KHashSearchIndexType) //需要指定PrefixExtractor
// 	//rocksOptions.SetBlockBasedTableFactory(blockBasedTableOptions)
// 	// rocksOptions.SetCompression(gorocksdb.LZ4Compression) //默认是Snappy，LZ4更快但是需要更多的磁盘

// 	//PlainTable
// 	rocksOptions.SetAllowMmapReads(true)
// 	//keyLen: plain table has optimization for fix-sized keys, which can be specified via user_key_len.  Alternatively, you can pass set it 0 if your keys have variable lengths.
// 	//bloomBitsPerKey: the number of bits used for bloom filer per prefix. You may disable it by passing a zero.
// 	//hashTableRatio: the desired utilization of the hash table used for prefix hashing. hash_table_ratio = number of prefixes / #buckets in the hash table
// 	//indexSparseness: inside each prefix, need to build one index record for how many keys for binary search inside each hash bucket. For encoding type kPrefix, the value will be used when writing to determine an interval to rewrite the full key. It will also be used as a suggestion and satisfied when possible.
// 	rocksOptions.SetPlainTableFactory(0, 0, 0.75, 16)
// 	//verify_checksum: As we are storing data in tmpfs and care read performance a lot, checksum could be disabled.
// 	readOptions.SetVerifyChecksums(false)
// 	rocksOptions.SetCompression(gorocksdb.SnappyCompression) //PlainTable暂不支持数据压缩

// 	rocksOptions.SetLevelCompactionDynamicLevelBytes(true)

// 	db, err := gorocksdb.OpenDb(rocksOptions, dbPath)
// 	if err != nil {
// 		panic(err)
// 	}

// 	return &Rocksdb{db: db, path: dbPath}, err
// }

// func (s *Rocksdb) GetPath() string {
// 	return s.path
// }

// func (s *Rocksdb) Set(k, v []byte) error {
// 	return s.db.Put(writeOptions, k, v)
// }

// func (s *Rocksdb) BatchSet(keys, values [][]byte) error {
// 	wb := gorocksdb.NewWriteBatch()
// 	defer wb.Destroy()
// 	for i, key := range keys {
// 		value := values[i]
// 		wb.Put(key, value)
// 	}
// 	s.db.Write(writeOptions, wb)
// 	return nil
// }

// // Get random Get is relatively slow in rocksdb
// func (s *Rocksdb) Get(k []byte) ([]byte, error) {
// 	return s.db.GetBytes(readOptions, k)
// }

// // Deprecated
// // BatchGet2 不保证顺序。MultiGet并不会比顺序调用GetBytes更快
// func (s *Rocksdb) BatchGet2(keys [][]byte) ([][]byte, error) {
// 	var slices gorocksdb.Slices
// 	var err error
// 	slices, err = s.db.MultiGet(readOptions, keys...)
// 	values := make([][]byte, 0, len(slices))
// 	if err == nil {
// 		for _, slice := range slices {
// 			values = append(values, slice.Data())
// 		}
// 		return values, nil
// 	}
// 	return values, err
// }

// // OverloadProtection 如果负载过高，则返回严重等级
// func (s *Rocksdb) OverloadProtection(ReadCount int) int {
// 	RealReadCount := ReadCount
// 	if time.Since(s.timeWindowBegin).Seconds() > 600 { //时间窗口设为1分钟
// 		s.timeWindowBegin = time.Now()
// 		atomic.StoreInt32(&s.load, 0)
// 	} else {
// 		if atomic.LoadInt32(&s.load) > 90000 { //1分钟单个DB读取docs数超过90000就拒绝任何请求
// 			log.Println("rocks overload reject any request")
// 			RealReadCount = 0
// 		} else if atomic.LoadInt32(&s.load) > 60000 { //1分钟单个DB读取docs数超过60000就只读20%
// 			log.Println("rocks overload reject 80% request")
// 			RealReadCount = ReadCount / 5
// 		} else if atomic.LoadInt32(&s.load) > 30000 { //1分钟单个DB读取docs数超过30000就只读50%
// 			log.Println("rocks overload reject 50% request")
// 			RealReadCount = ReadCount / 2
// 		}
// 	}
// 	atomic.AddInt32(&s.load, int32(RealReadCount))
// 	return RealReadCount
// }

// // BatchGet 不保证顺序，MultiGet并不会比顺序调用GetBytes更快，所以采用并发调用GetBytes
// func (s *Rocksdb) BatchGet(keys [][]byte) ([][]byte, error) {
// 	if len(keys) > 0 {
// 		RealReadCount := len(keys)
// 		if RealReadCount > 10 {
// 			//过载保护
// 			RealReadCount = s.OverloadProtection(len(keys))
// 		}
// 		if RealReadCount <= 0 {
// 			return [][]byte{}, nil
// 		}
// 		values := make([][]byte, RealReadCount)

// 		const READ_SEG_LEN = 500
// 		segCount := (RealReadCount + READ_SEG_LEN - 1) / READ_SEG_LEN
// 		wg := sync.WaitGroup{}
// 		wg.Add(segCount)
// 		for segIndex := 0; segIndex < segCount; segIndex++ {
// 			go func(segIndex int) {
// 				defer wg.Done()
// 				begin := segIndex * READ_SEG_LEN
// 				end := (segIndex + 1) * READ_SEG_LEN
// 				for i := begin; i < RealReadCount && i < end; i++ {
// 					key := keys[i]
// 					if value, err := s.db.GetBytes(readOptions, key); err == nil {
// 						values[i] = value
// 					} else {
// 						values[i] = []byte{}
// 					}
// 				}
// 			}(segIndex)
// 		}
// 		wg.Wait()
// 		return values, nil
// 	} else {
// 		return [][]byte{}, nil
// 	}
// }

// func (s *Rocksdb) Delete(k []byte) error {
// 	return s.db.Delete(writeOptions, k)
// }

// func (s *Rocksdb) BatchDelete(keys [][]byte) error {
// 	wb := gorocksdb.NewWriteBatch()
// 	defer wb.Destroy()
// 	for _, key := range keys {
// 		wb.Delete(key)
// 	}
// 	s.db.Write(writeOptions, wb)
// 	return nil
// }

// // Has
// func (s *Rocksdb) Has(k []byte) bool {
// 	values, err := s.db.GetBytes(readOptions, k)
// 	if err == nil && len(values) > 0 {
// 		return true
// 	}
// 	return false
// }

// func (s *Rocksdb) IterDB(fn func(k, v []byte) error) int64 {
// 	var total int64
// 	iter := s.db.NewIterator(readOptions)
// 	defer iter.Close()
// 	for iter.SeekToFirst(); iter.Valid(); iter.Next() {
// 		// fmt.Println("next")
// 		//k := make([]byte, 4)
// 		//copy(k, iter.Key().Data())
// 		//value := iter.Value().Data()
// 		//v := make([]byte, len(value))
// 		//copy(v, value)
// 		//fn(k, v)
// 		if err := fn(iter.Key().Data(), iter.Value().Data()); err == nil {
// 			atomic.AddInt64(&total, 1)
// 		}
// 	}
// 	return atomic.LoadInt64(&total)
// }

// func (s *Rocksdb) IterKey(fn func(k []byte) error) int64 {
// 	var total int64
// 	iter := s.db.NewIterator(readOptions)
// 	defer iter.Close()
// 	for iter.SeekToFirst(); iter.Valid(); iter.Next() {
// 		//k := make([]byte, 4)
// 		//copy(k, iter.Key().Data())
// 		//fn(k)
// 		if err := fn(iter.Key().Data()); err == nil {
// 			atomic.AddInt64(&total, 1)
// 		}
// 	}
// 	return atomic.LoadInt64(&total)
// }

// func (s *Rocksdb) Close() error {
// 	s.db.Close()
// 	return nil
// }
