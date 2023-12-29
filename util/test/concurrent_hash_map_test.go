package test

import (
	"math/rand"
	"sync"
	"testing"

	"github.com/Orisun/radic/v2/util"
)

var conMp = util.NewConcurrentHashMap[int64](8, 1000)
var synMp = sync.Map{}

func readConMap() {
	for i := 0; i < 1000; i++ {
		key := rand.Int63()
		conMp.Get(key)
	}
}

func writeConMap() {
	for i := 0; i < 1000; i++ {
		key := rand.Int63()
		conMp.Set(key, 1)
	}
}

func readSynMap() {
	for i := 0; i < 1000; i++ {
		key := rand.Int63()
		synMp.Load(key)
	}
}

func writeSynMap() {
	for i := 0; i < 1000; i++ {
		key := rand.Int63()
		synMp.Store(key, 1)
	}
}

func BenchmarkConMap(b *testing.B) {
	for i := 0; i < b.N; i++ {
		const P = 300
		wg := sync.WaitGroup{}
		wg.Add(2 * P)
		for i := 0; i < P; i++ { //300个协程一直读
			go func() {
				defer wg.Done()
				for i := 0; i < 10; i++ {
					readConMap()
				}
			}()
		}
		for i := 0; i < P; i++ { //300个协程一直写
			go func() {
				defer wg.Done()
				for i := 0; i < 10; i++ {
					writeConMap()
					// time.Sleep(100 * time.Millisecond)   //写很少时速度差1倍，一直写时速度差3倍
				}
			}()
		}
		wg.Wait()
	}
}

func BenchmarkSynMap(b *testing.B) {
	for i := 0; i < b.N; i++ {
		const P = 300
		wg := sync.WaitGroup{}
		wg.Add(2 * P)
		for i := 0; i < P; i++ { //300个协程一直读
			go func() {
				defer wg.Done()
				for i := 0; i < 10; i++ {
					readSynMap()
				}
			}()
		}
		for i := 0; i < P; i++ { //300个协程一直写
			go func() {
				defer wg.Done()
				for i := 0; i < 10; i++ {
					writeSynMap()
					// time.Sleep(100 * time.Millisecond)
				}
			}()
		}
		wg.Wait()
	}
}

// go test ./util/test -bench=Map -run=^$ -count=1 -benchmem -benchtime=3s
/*
goos: windows
goarch: amd64
pkg: gift/util/test
cpu: 11th Gen Intel(R) Core(TM) i7-1165G7 @ 2.80GHz
BenchmarkConMap-8              4        1174202475 ns/op        632463264 B/op  18081067 allocs/op
BenchmarkSynMap-8              1        3449703900 ns/op        433304672 B/op  12091765 allocs/op
*/
