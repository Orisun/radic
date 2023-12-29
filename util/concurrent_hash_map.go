package util

import (
	"sync"
	"unsafe"

	farmhash "github.com/leemcloughlin/gofarmhash"
)

// 自行实现支持并发读写的map
type ConcurrentHashMap[T comparable] struct {
	mps   []map[T]any    //由多个小map构成
	seg   int            //小map的个数
	locks []sync.RWMutex //每个小map配一把读写锁。避免全局只有一把锁，影响性能
	seed  uint32         //每次执行farmhash传统一的seed
}

// cap预估map中容纳多少元素，seg内部包含几个小map
func NewConcurrentHashMap[T comparable](seg, cap int) *ConcurrentHashMap[T] {
	mps := make([]map[T]any, seg)
	locks := make([]sync.RWMutex, seg)
	for i := 0; i < seg; i++ {
		mps[i] = make(map[T]any, cap/seg)
		locks[i] = sync.RWMutex{}
	}
	return &ConcurrentHashMap[T]{
		mps:   mps,
		seg:   seg,
		seed:  0,
		locks: locks,
	}
}

// 指针转int
func Pointer2Int[T comparable](p *T) int {
	return *(*int)(unsafe.Pointer(p))
}

// 判断key对应到哪个小map
func (m *ConcurrentHashMap[T]) getSegIndex(key T) int {
	hash := int(farmhash.Hash32WithSeed(IntToBytes(Pointer2Int(&key)), m.seed))
	return hash % m.seg
}

// 写入<key, value>
func (m *ConcurrentHashMap[T]) Set(key T, value any) {
	index := m.getSegIndex(key)
	m.locks[index].Lock()
	defer m.locks[index].Unlock()
	m.mps[index][key] = value
}

// 根据key读取value
func (m *ConcurrentHashMap[T]) Get(key T) (any, bool) {
	index := m.getSegIndex(key)
	m.locks[index].RLock()
	defer m.locks[index].RUnlock()
	value, exists := m.mps[index][key]
	return value, exists
}
