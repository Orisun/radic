package test

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/Orisun/radic/v2/index_service"
)

var (
	balancer  index_service.LoadBalancer
	endpoints = []string{"127.0.0.1", "127.0.0.2", "127.0.0.3"}
)

func testLB(balancer index_service.LoadBalancer) {
	const P = 100 //开100个协程并发使用balancer
	const LOOP = 100
	selected := make(chan string, P*LOOP)
	wg := sync.WaitGroup{}
	wg.Add(P)
	for i := 0; i < P; i++ {
		go func() {
			defer wg.Done()
			for i := 0; i < LOOP; i++ {
				endpoint := balancer.Take(endpoints)                        //取出一个endpoint
				selected <- endpoint                                        //为什么不直接使用一个sync.Map进行计数？
				time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond) //假装在使用endpoint
			}
		}()
	}
	wg.Wait()
	close(selected)

	cm := make(map[string]int, len(endpoints))
	for {
		endpoint, ok := <-selected
		if !ok {
			break
		}
		value, ok := cm[endpoint]
		if ok {
			cm[endpoint] = value + 1
		} else {
			cm[endpoint] = 1
		}
	}

	for k, v := range cm {
		fmt.Println(k, v) //打印每个endpoint被使用了几次
	}
}

func TestRandomSelect(t *testing.T) {
	balancer = new(index_service.RandomSelect)
	testLB(balancer)
}

func TestRoudRobin(t *testing.T) {
	balancer = new(index_service.RoundRobin)
	testLB(balancer)
}

// go test -v ./index_service/test -run=^TestRandomSelect$ -count=1
// go test -v ./index_service/test -run=^TestRoudRobin$ -count=1
