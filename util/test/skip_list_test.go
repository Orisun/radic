package test

import (
	"fmt"
	"testing"

	"github.com/huandu/skiplist"
)

func TestSkipList(t *testing.T) {
	list := skiplist.New(skiplist.Int32)
	list.Set(24, 31) //skiplist是一个按key排序好的map
	list.Set(24, 40) //相同的key, value会覆盖前值
	list.Set(12, 40) //添加元素
	list.Set(18, 3)
	list.Remove(12)                         //删除元素
	if value, ok := list.GetValue(18); ok { //查找key对应的value
		fmt.Println(value)
	}
	//遍历。自动按key排好序
	fmt.Println("------------------")
	node := list.Front()
	for node != nil {
		fmt.Println(node.Key(), node.Value)
		node = node.Next() //迭代器模式
	}
}

// go test -v ./util/test -run=^TestSkipList$ -count=1
