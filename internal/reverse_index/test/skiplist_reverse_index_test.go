package test

import (
	"fmt"
	"strings"
	"testing"

	reverseindex "github.com/Orisun/radic/v2/internal/reverse_index"
	"github.com/huandu/skiplist"
)

func TestIntersectionOfSkipList(t *testing.T) {
	l1 := skiplist.New(skiplist.Uint32)
	l1.Set(uint32(5), 0)
	l1.Set(uint32(1), 0)
	l1.Set(uint32(4), 0)
	l1.Set(uint32(9), 0)
	l1.Set(uint32(11), 0)
	l1.Set(uint32(7), 0)
	//skiplist内部会自动做排序，排完序之后为 1 4 5 7 9 11

	l2 := skiplist.New(skiplist.Uint32)
	l2.Set(uint32(4), 0)
	l2.Set(uint32(5), 0)
	l2.Set(uint32(9), 0)
	l2.Set(uint32(8), 0)
	l2.Set(uint32(2), 0)
	//skiplist内部会自动做排序，排完序之后为 2 4 5 8 9

	l3 := skiplist.New(skiplist.Uint32)
	l3.Set(uint32(3), 0)
	l3.Set(uint32(5), 0)
	l3.Set(uint32(7), 0)
	l3.Set(uint32(9), 0)
	//skiplist内部会自动做排序，排完序之后为 3 5 7 9

	interset := reverseindex.IntersectionOfSkipList()
	if interset != nil {
		node := interset.Front()
		for node != nil {
			fmt.Printf("%d ", node.Key().(uint32))
			node = node.Next()
		}
	}
	fmt.Println("\n" + strings.Repeat("-", 50))

	interset = reverseindex.IntersectionOfSkipList(l1)
	if interset != nil {
		node := interset.Front()
		for node != nil {
			fmt.Printf("%d ", node.Key().(uint32))
			node = node.Next()
		}
	}
	fmt.Println("\n" + strings.Repeat("-", 50))

	interset = reverseindex.IntersectionOfSkipList(l1, l2)
	if interset != nil {
		node := interset.Front()
		for node != nil {
			fmt.Printf("%d ", node.Key().(uint32))
			node = node.Next()
		}
	}
	fmt.Println("\n" + strings.Repeat("-", 50))

	interset = reverseindex.IntersectionOfSkipList(l1, l2, l3)
	if interset != nil {
		node := interset.Front()
		for node != nil {
			fmt.Printf("%d ", node.Key().(uint32))
			node = node.Next()
		}
	}
	fmt.Println("\n" + strings.Repeat("-", 50))

	union := reverseindex.UnionsetOfSkipList()
	if union != nil {
		node := union.Front()
		for node != nil {
			fmt.Printf("%d ", node.Key().(uint32))
			node = node.Next()
		}
	}
	fmt.Println("\n" + strings.Repeat("-", 50))

	union = reverseindex.UnionsetOfSkipList(l1)
	if union != nil {
		node := union.Front()
		for node != nil {
			fmt.Printf("%d ", node.Key().(uint32))
			node = node.Next()
		}
	}
	fmt.Println("\n" + strings.Repeat("-", 50))

	union = reverseindex.UnionsetOfSkipList(l1, l2)
	if union != nil {
		node := union.Front()
		for node != nil {
			fmt.Printf("%d ", node.Key().(uint32))
			node = node.Next()
		}
	}
	fmt.Println("\n" + strings.Repeat("-", 50))

	union = reverseindex.UnionsetOfSkipList(l1, l2, l3)
	if union != nil {
		node := union.Front()
		for node != nil {
			fmt.Printf("%d ", node.Key().(uint32))
			node = node.Next()
		}
	}
	fmt.Println("\n" + strings.Repeat("-", 50))
}

//  go test -v .\internal\reverse_index\test -run=^TestIntersectionOfSkipList$ -count=1
