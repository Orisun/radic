// Date         : 2019-04-13 19:52
// Author       : zhangzhaoyang
// Description  :

package util

import (
	"encoding/json"
	"sync"
)

// A "thread" safe map of type uint32:Anything.
// To avoid lock bottlenecks this map is dived to several (DEFAULT_SHARD_COUNT) map shards.
type ConcurrentMapUint32 struct {
	tables      []*concurrentMapSharedUint32
	shard_count int
}

// A "thread" safe uint32 to anything map.
type concurrentMapSharedUint32 struct {
	items        map[uint32]interface{}
	sync.RWMutex // Read Write mutex, guards access to internal map.
}

// Creates a new concurrent map.
func NewConcurrentMapUint32(shardCount int) *ConcurrentMapUint32 {
	if shardCount <= 0 {
		shardCount = DEFAULT_SHARD_COUNT
	}
	rect := ConcurrentMapUint32{
		shard_count: shardCount,
	}
	m := make([]*concurrentMapSharedUint32, shardCount)
	for i := 0; i < shardCount; i++ {
		m[i] = &concurrentMapSharedUint32{items: make(map[uint32]interface{})}
	}
	rect.tables = m
	return &rect
}

// Returns shard under given key
func (m *ConcurrentMapUint32) GetShard(key uint32) *concurrentMapSharedUint32 {
	return m.tables[uint(fnvuint32(key))%uint(m.shard_count)]
}

func (m *ConcurrentMapUint32) MSet(data map[uint32]interface{}) {
	for key, value := range data {
		shard := m.GetShard(key)
		shard.Lock()
		shard.items[key] = value
		shard.Unlock()
	}
}

// Sets the given value under the specified key.
func (m *ConcurrentMapUint32) Set(key uint32, value interface{}) {
	// Get map shard.
	shard := m.GetShard(key)
	shard.Lock()
	shard.items[key] = value
	shard.Unlock()
}

// Insert or Update - updates existing element or inserts a new one using UpsertCb
func (m *ConcurrentMapUint32) Upsert(key uint32, value interface{}, cb UpsertCb) (res interface{}) {
	shard := m.GetShard(key)
	shard.Lock()
	v, ok := shard.items[key]
	res = cb(ok, v, value)
	shard.items[key] = res
	shard.Unlock()
	return res
}

// Sets the given value under the specified key if no value was associated with it.
func (m *ConcurrentMapUint32) SetIfAbsent(key uint32, value interface{}) bool {
	// Get map shard.
	shard := m.GetShard(key)
	shard.Lock()
	_, ok := shard.items[key]
	if !ok {
		shard.items[key] = value
	}
	shard.Unlock()
	return !ok
}

func (m *ConcurrentMapUint32) Inr(key uint32, delta int) bool {
	// Get map shard.
	shard := m.GetShard(key)
	shard.Lock()
	v, ok := shard.items[key]
	if !ok {
		shard.items[key] = delta
	} else {
		shard.items[key] = v.(int) + delta
	}
	shard.Unlock()
	return !ok
}

// Retrieves an element from map under given key.
func (m *ConcurrentMapUint32) Get(key uint32) (interface{}, bool) {
	// Get shard
	shard := m.GetShard(key)
	shard.RLock()
	// Get item from shard.
	val, ok := shard.items[key]
	shard.RUnlock()
	return val, ok
}

// Returns the number of elements within the map.
func (m *ConcurrentMapUint32) Count() int {
	count := 0
	for i := 0; i < m.shard_count; i++ {
		shard := m.tables[i]
		shard.RLock()
		count += len(shard.items)
		shard.RUnlock()
	}
	return count
}

// Looks up an item under specified key
func (m *ConcurrentMapUint32) Has(key uint32) bool {
	// Get shard
	shard := m.GetShard(key)
	shard.RLock()
	// See if element is within shard.
	_, ok := shard.items[key]
	shard.RUnlock()
	return ok
}

// Removes an element from the map.
func (m *ConcurrentMapUint32) Remove(key uint32) {
	// Try to get shard.
	shard := m.GetShard(key)
	shard.Lock()
	delete(shard.items, key)
	shard.Unlock()
}

// Removes an element from the map and returns it
func (m *ConcurrentMapUint32) Pop(key uint32) (v interface{}, exists bool) {
	// Try to get shard.
	shard := m.GetShard(key)
	shard.Lock()
	v, exists = shard.items[key]
	delete(shard.items, key)
	shard.Unlock()
	return v, exists
}

// Checks if map is empty.
func (m *ConcurrentMapUint32) IsEmpty() bool {
	return m.Count() == 0
}

// Used by the Iter & IterBuffered functions to wrap two variables together over a channel,
type TupleUint32 struct {
	Key uint32
	Val interface{}
}

// Returns an iterator which could be used in a for range loop.
//
// Deprecated: using IterBuffered() will get a better performence
func (m *ConcurrentMapUint32) Iter() <-chan TupleUint32 {
	chans := snapshotUint32(m)
	ch := make(chan TupleUint32)
	go fanInuint32(chans, ch)
	return ch
}

// Returns a buffered iterator which could be used in a for range loop.
func (m *ConcurrentMapUint32) IterBuffered() <-chan TupleUint32 {
	chans := snapshotUint32(m)
	total := 0
	for _, c := range chans {
		total += cap(c)
	}
	ch := make(chan TupleUint32, total)
	go fanInuint32(chans, ch)
	return ch
}

// Returns a array of channels that contains elements in each shard,
// which likely takes a snapshotUint32 of `m`.
// It returns once the size of each buffered channel is determined,
// before all the channels are populated using goroutines.
func snapshotUint32(m *ConcurrentMapUint32) (chans []chan TupleUint32) {
	chans = make([]chan TupleUint32, m.shard_count)
	wg := sync.WaitGroup{}
	wg.Add(m.shard_count)
	// Foreach shard.
	for index, shard := range m.tables {
		go func(index int, shard *concurrentMapSharedUint32) { //注意：在子协程中使用for range生成的变量时一定作为参数传给子协程
			// Foreach key, value pair.
			shard.RLock()
			chans[index] = make(chan TupleUint32, len(shard.items))
			wg.Done()
			for key, val := range shard.items {
				chans[index] <- TupleUint32{key, val}
			}
			shard.RUnlock()
			close(chans[index])
		}(index, shard)
	}
	wg.Wait()
	return chans
}

// fanInuint32 reads elements from channels `chans` into channel `out`
func fanInuint32(chans []chan TupleUint32, out chan TupleUint32) {
	wg := sync.WaitGroup{}
	wg.Add(len(chans))
	for _, ch := range chans {
		go func(ch chan TupleUint32) { //注意：在子协程中使用for range生成的变量时一定作为参数传给子协程
			for t := range ch {
				out <- t
			}
			wg.Done()
		}(ch)
	}
	wg.Wait()
	close(out)
}

// Returns all items as map[uint32]interface{}
func (m *ConcurrentMapUint32) Items() map[uint32]interface{} {
	tmp := make(map[uint32]interface{})

	// Insert items to temporary map.
	for item := range m.IterBuffered() {
		tmp[item.Key] = item.Val
	}

	return tmp
}

// Iterator callback,called for every key,value found in
// maps. RLock is held for all calls for a given shard
// therefore callback sess consistent view of a shard,
// but not across the shards
type IterCbUint32 func(key uint32, v interface{})

// Callback based iterator, cheapest way to read
// all elements in a map.
func (m *ConcurrentMapUint32) IterCb(fn IterCbUint32) {
	for idx := range m.tables {
		shard := (m.tables)[idx]
		shard.RLock()
		for key, value := range shard.items {
			fn(key, value)
		}
		shard.RUnlock()
	}
}

// Return all keys as []uint32
func (m *ConcurrentMapUint32) Keys() []uint32 {
	count := m.Count()
	ch := make(chan uint32, count)
	go func() {
		// 遍历所有的 shard.
		wg := sync.WaitGroup{}
		wg.Add(m.shard_count)
		for _, shard := range m.tables {
			go func(shard *concurrentMapSharedUint32) { //注意：在子协程中使用for range生成的变量时一定作为参数传给子协程
				// 遍历所有的 key, value 键值对.
				shard.RLock()
				for key := range shard.items {
					ch <- key
				}
				shard.RUnlock()
				wg.Done()
			}(shard)
		}
		wg.Wait()
		close(ch)
	}()

	// 生成 keys 数组，存储所有的 key
	keys := make([]uint32, 0, count)
	for k := range ch {
		keys = append(keys, k)
	}
	return keys
}

//Reviles ConcurrentMapUint32 "private" variables to json marshal.
func (m *ConcurrentMapUint32) MarshalJSON() ([]byte, error) {
	// Create a temporary map, which will hold all item spread across shards.
	tmp := make(map[uint32]interface{})

	// Insert items to temporary map.
	for item := range m.IterBuffered() {
		tmp[item.Key] = item.Val
	}
	return json.Marshal(tmp)
}

func fnvuint32(key uint32) uint32 {
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	hash *= prime32
	hash ^= key
	return hash
}
