package course

import (
	"container/ring"
	"fmt"
	"slices"
)

func TraverseRing(ring *ring.Ring) {
	ring.Do(func(i interface{}) { //通过Do()来遍历Ring。函数参数用来指定如何处理Ring里的元素
		fmt.Printf("%v ", i)
	})
	fmt.Println()
}

func compare1(a, b *Doc) int {
	return a.Id - b.Id
}

func compare2(a, b *Doc) int {
	return b.Id - a.Id
}

func SortDoc1(docs []*Doc, compare func(a, b *Doc) int) {
	slices.SortFunc(docs, compare)
}

type DocComparatorFunc func(a, b *Doc) int //把函数搞成一个type，看着更顺眼

func SortDoc2(docs []*Doc, compare DocComparatorFunc) {
	slices.SortFunc(docs, compare)
}

type IDocComparator interface { //用接口替代函数参数
	Compare(a, b *Doc) int
}

type PositiveOrder struct{}

func (PositiveOrder) Compare(a, b *Doc) int {
	return a.Id - b.Id //正序
}

type ReversedOrder struct{}

func (ReversedOrder) Compare(a, b *Doc) int {
	return b.Id - a.Id //逆序
}

func SortDoc3(docs []*Doc, comparator IDocComparator) {
	slices.SortFunc(docs, comparator.Compare)
}

func funcArgVsInterface() {
	docs := make([]*Doc, 10)
	SortDoc1(docs, compare1) //函数作为参数
	SortDoc1(docs, compare2)
	SortDoc2(docs, compare1) //函数作为参数
	SortDoc2(docs, compare2)
	SortDoc3(docs, PositiveOrder{}) //接口作为参数
	SortDoc3(docs, ReversedOrder{})
}
