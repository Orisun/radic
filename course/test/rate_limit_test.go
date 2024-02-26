package test

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Orisun/radic/v2/course"
)

func TestLimit(t *testing.T) {
	go course.CallHandler()
	for i := 0; i < 5; i++ {
		time.Sleep(time.Second)
		fmt.Printf("过去1秒钟接口调用了%d次\n", atomic.LoadInt32(&course.TotalQuery))
		atomic.StoreInt32(&course.TotalQuery, 0) //每隔一秒，清0一次
	}
}

// go test -v ./course/test -run=^TestLimit$ -count=1
