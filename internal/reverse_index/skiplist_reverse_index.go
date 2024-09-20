package reverseindex

import (
	"runtime"
	"sync"

	"github.com/Orisun/radic/v2/types"
	"github.com/Orisun/radic/v2/util"
	"github.com/huandu/skiplist"
	farmhash "github.com/leemcloughlin/gofarmhash"
)

// 倒排索引整体上是个map，map的value是一个List
type SkipListReverseIndex struct {
	table *util.ConcurrentHashMap //分段map，并发安全
	locks []sync.RWMutex          //修改倒排索引时，相同的key需要去竞争同一把锁
}

// DocNumEstimate是预估的doc数量
func NewSkipListReverseIndex(DocNumEstimate int) *SkipListReverseIndex {
	indexer := new(SkipListReverseIndex)
	indexer.table = util.NewConcurrentHashMap(runtime.NumCPU(), DocNumEstimate)
	indexer.locks = make([]sync.RWMutex, 1000)
	return indexer
}

func (indexer SkipListReverseIndex) getLock(key string) *sync.RWMutex {
	n := int(farmhash.Hash32WithSeed([]byte(key), 0))
	return &indexer.locks[n%len(indexer.locks)]
}

// 倒排链上的每个元素包含2部分：SkipListKey（即Document.IntId）和SkipListValue（即Document.Id和Document.BitsFeature）
type SkipListValue struct {
	Id          string
	BitsFeature uint64
}

// 添加一个doc
func (indexer *SkipListReverseIndex) Add(doc types.Document) {
	for _, keyword := range doc.Keywords {
		key := keyword.ToString()
		lock := indexer.getLock(key)
		lock.Lock() //准备修改倒排链，加写锁
		sklValue := SkipListValue{doc.Id, doc.BitsFeature}
		if value, exists := indexer.table.Get(key); exists {
			list := value.(*skiplist.SkipList)
			list.Set(doc.IntId, sklValue) //IntId作为SkipList的key，而value里则包含了业务侧的文档id和BitsFeature
		} else {
			list := skiplist.New(skiplist.Uint64)
			list.Set(doc.IntId, sklValue)
			indexer.table.Set(key, list)
		}
		// util.Log.Printf("add key %s value %d to reverse index\n", key, DocId)
		lock.Unlock()
	}
}

// 从key上删除对应的doc
func (indexer *SkipListReverseIndex) Delete(IntId uint64, keyword *types.Keyword) {
	key := keyword.ToString()
	lock := indexer.getLock(key)
	lock.Lock() //准备修改倒排链，加写锁
	if value, exists := indexer.table.Get(key); exists {
		list := value.(*skiplist.SkipList)
		list.Remove(IntId)
	}
	lock.Unlock()
}

// 求多个SkipList的交集
func IntersectionOfSkipList(lists ...*skiplist.SkipList) *skiplist.SkipList {
	if len(lists) == 0 {
		return nil
	}
	if len(lists) == 1 {
		return lists[0]
	}
	result := skiplist.New(skiplist.Uint64)
	currNodes := make([]*skiplist.Element, len(lists)) //给每条SkipList分配一个指针，从前往后遍历
	for i, list := range lists {
		if list == nil || list.Len() == 0 { //只要lists中有一条是空链，则交集为空
			return nil
		}
		currNodes[i] = list.Front()
	}
	for {
		maxList := make(map[int]struct{}, len(currNodes)) //此刻，哪个指针对应的值最大（最大者可能存在多个，所以用map）
		var maxValue uint64 = 0
		for i, node := range currNodes {
			if node.Key().(uint64) > maxValue {
				maxValue = node.Key().(uint64)
				maxList = map[int]struct{}{i: {}} //清空之前的map，新map里面放入一个元素i。可以用一对大括号表示空结构体实例
			} else if node.Key().(uint64) == maxValue {
				maxList[i] = struct{}{}
			}
		}
		if len(maxList) == len(currNodes) { //所有node的值都一样大，则新诞生一个交集
			result.Set(currNodes[0].Key(), currNodes[0].Value)
			for i, node := range currNodes { //所有node均需往后移
				currNodes[i] = node.Next()
				if currNodes[i] == nil {
					return result
				}
			}
		} else {
			for i, node := range currNodes {
				if _, exists := maxList[i]; !exists { //值大的不动，小的往后移
					currNodes[i] = node.Next() //不能用node=node.Next()，因为for range取得的是值拷贝
					if currNodes[i] == nil {   //只要有一条SkipList已走到最后，则说明不会再有新的交集诞生，可以return了
						return result
					}
				}
			}
		}
	}
}

// 求多个SkipList的并集
func UnionsetOfSkipList(lists ...*skiplist.SkipList) *skiplist.SkipList {
	if len(lists) == 0 {
		return nil
	}
	if len(lists) == 1 {
		return lists[0]
	}
	result := skiplist.New(skiplist.Uint64)
	for _, list := range lists {
		if list == nil {
			continue
		}
		node := list.Front()
		for node != nil {
			result.Set(node.Key(), node.Value)
			node = node.Next()
		}
	}
	return result
}

// 按照bits特征进行过滤
func (indexer SkipListReverseIndex) FilterByBits(bits uint64, onFlag uint64, offFlag uint64, orFlags []uint64) bool {
	//onFlag所有bit必须全部命中
	if bits&onFlag != onFlag {
		return false
	}
	//offFlag所有bit必须全部不命中
	if bits&offFlag != 0 {
		return false
	}
	//多个orFlags必须全部命中
	for _, orFlag := range orFlags {
		if orFlag > 0 && bits&orFlag <= 0 { //单个orFlag只人有一个bit命中即可
			return false
		}
	}
	return true
}

// 搜索，返回SkipList
func (indexer SkipListReverseIndex) search(q *types.TermQuery, onFlag uint64, offFlag uint64, orFlags []uint64) *skiplist.SkipList {
	if q.Keyword != nil {
		Keyword := q.Keyword.ToString()
		lock := indexer.getLock(Keyword)
		lock.RLock() //准备遍历倒排链，加读锁
		defer lock.RUnlock()
		if value, exists := indexer.table.Get(Keyword); exists {
			result := skiplist.New(skiplist.Uint64)
			list := value.(*skiplist.SkipList)
			// util.Log.Printf("retrive %d docs by key %s", list.Len(), Keyword)
			node := list.Front()
			for node != nil {
				intId := node.Key().(uint64)
				skv, _ := node.Value.(SkipListValue)
				flag := skv.BitsFeature
				if intId > 0 && indexer.FilterByBits(flag, onFlag, offFlag, orFlags) { //确保有效元素都大于0
					result.Set(intId, skv)
				}
				node = node.Next()
			}
			return result
		}
	} else if len(q.Must) > 0 {
		results := make([]*skiplist.SkipList, 0, len(q.Must))
		for _, q := range q.Must {
			results = append(results, indexer.search(q, onFlag, offFlag, orFlags))
		}
		return IntersectionOfSkipList(results...)
	} else if len(q.Should) > 0 {
		results := make([]*skiplist.SkipList, 0, len(q.Should))
		for _, q := range q.Should {
			results = append(results, indexer.search(q, onFlag, offFlag, orFlags))
		}
		return UnionsetOfSkipList(results...)
	}
	return nil
}

// 搜索，返回docId
func (indexer SkipListReverseIndex) Search(query *types.TermQuery, onFlag uint64, offFlag uint64, orFlags []uint64) []string {
	result := indexer.search(query, onFlag, offFlag, orFlags)
	if result == nil {
		return nil
	}
	arr := make([]string, 0, result.Len())
	node := result.Front()
	for node != nil {
		skv, _ := node.Value.(SkipListValue)
		arr = append(arr, skv.Id)
		node = node.Next()
	}
	return arr
}
