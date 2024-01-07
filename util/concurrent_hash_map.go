package util

import (
	"sync"

	farmhash "github.com/leemcloughlin/gofarmhash"
)

// 自行实现支持并发读写的map。key是string，value是any
type ConcurrentHashMap struct {
	mps   []map[string]any //由多个小map构成
	seg   int              //小map的个数
	locks []sync.RWMutex   //每个小map配一把读写锁。避免全局只有一把锁，影响性能
	seed  uint32           //每次执行farmhash传统一的seed
}

// cap预估map中容纳多少元素，seg内部包含几个小map
func NewConcurrentHashMap(seg, cap int) *ConcurrentHashMap {
	mps := make([]map[string]any, seg)
	locks := make([]sync.RWMutex, seg)
	for i := 0; i < seg; i++ {
		mps[i] = make(map[string]any, cap/seg)
		locks[i] = sync.RWMutex{}
	}
	return &ConcurrentHashMap{
		mps:   mps,
		seg:   seg,
		seed:  0,
		locks: locks,
	}
}

// 判断key对应到哪个小map
func (m *ConcurrentHashMap) getSegIndex(key string) int {
	hash := int(farmhash.Hash32WithSeed([]byte(key), m.seed))
	return hash % m.seg
}

// 写入<key, value>
func (m *ConcurrentHashMap) Set(key string, value any) {
	index := m.getSegIndex(key)
	m.locks[index].Lock()
	defer m.locks[index].Unlock()
	m.mps[index][key] = value
}

// 根据key读取value
func (m *ConcurrentHashMap) Get(key string) (any, bool) {
	index := m.getSegIndex(key)
	m.locks[index].RLock()
	defer m.locks[index].RUnlock()
	value, exists := m.mps[index][key]
	return value, exists
}

func (m *ConcurrentHashMap) Iter(f func(key string, value any)) {
	for _, mp := range m.mps {
		for k, v := range mp {
			f(k, v)
		}
	}
}
