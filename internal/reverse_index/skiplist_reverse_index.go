package reverseindex

import (
	"runtime"
	"sync"

	"github.com/Orisun/radic/v2/util"
	"github.com/huandu/skiplist"
	farmhash "github.com/leemcloughlin/gofarmhash"
)

// 倒排索引整体上是个map，map的value是一个List
type SkipListReverseIndex struct {
	table *util.ConcurrentHashMap[string] //分段map，并发安全
	locks []sync.RWMutex                  //修改倒排索引时，相同的key需要去竞争同一把锁
}

// DocNumEstimate是预估的doc数量
func NewSkipListReverseIndex(DocNumEstimate int) *SkipListReverseIndex {
	indexer := new(SkipListReverseIndex)
	indexer.table = util.NewConcurrentHashMap[string](runtime.NumCPU(), DocNumEstimate)
	indexer.locks = make([]sync.RWMutex, 1000)
	return indexer
}

func (indexer SkipListReverseIndex) getLock(key string) *sync.RWMutex {
	n := int(farmhash.Hash32WithSeed([]byte(key), 0))
	return &indexer.locks[n%len(indexer.locks)]
}

// 添加一个doc
func (indexer SkipListReverseIndex) Add(DocId uint32, FeatureBits uint32, Keywords []string) {
	for _, key := range Keywords {
		lock := indexer.getLock(key)
		lock.Lock()
		if value, exists := indexer.table.Get(key); exists {
			list := value.(skiplist.SkipList)
			list.Set(DocId, FeatureBits) //DocId作为SkipList的key，FeatureBits作为SkipList的Value
		} else {
			list := skiplist.New(skiplist.Int32)
			list.Set(DocId, FeatureBits)
			indexer.table.Set(key, list)
		}
		lock.Unlock()
	}
}

// 从key上删除对应的doc
func (indexer SkipListReverseIndex) Delete(DocId uint32, Keyword string) {
	lock := indexer.getLock(Keyword)
	lock.Lock()
	if value, exists := indexer.table.Get(Keyword); exists {
		list := value.(skiplist.SkipList)
		list.Remove(DocId)
	}
	lock.Unlock()
}
