package test

import (
	"fmt"
	"testing"

	"github.com/Orisun/radic/v2/course"
)

func TestBits(t *testing.T) {
	var n uint64
	n = course.SetBit1(n, 11)
	n = course.SetBit1(n, 28)

	fmt.Println(course.IsBit1(n, 11))
	fmt.Println(course.IsBit1(n, 28))
	fmt.Println(course.IsBit1(n, 18))

	fmt.Println(course.CountBit1(n))
	fmt.Printf("%064b\n", n)
}

func BenchmarkBitFilter1(b *testing.B) {
	candidate := &course.Candidate{}
	candidate.SetActive(5)
	candidate.SetVip()

	const LEN = 10
	lst := make([]*course.Candidate, 0, LEN)
	for i := 0; i < LEN; i++ {
		lst = append(lst, candidate)
	}

	male, vip, weekActive := true, false, true //搜索条件要求是男性、周活
	b.ResetTimer()                             //开始计时
	for i := 0; i < b.N; i++ {
		for _, candidate := range lst {
			candidate.Filter1(male, vip, weekActive)
		}
	}
}

func BenchmarkBitFilter2(b *testing.B) {
	candidate := &course.Candidate{}
	candidate.SetActive(5)
	candidate.SetVip()

	const LEN = 10
	lst := make([]*course.Candidate, 0, LEN)
	for i := 0; i < LEN; i++ {
		lst = append(lst, candidate)
	}

	var on uint64
	on |= course.MALE //搜索条件要求是男性、周活
	on |= course.WEEK_ACTIVE
	b.ResetTimer() //开始计时
	for i := 0; i < b.N; i++ {
		for _, candidate := range lst {
			candidate.Filter2(on)
		}
	}
}

func TestBitMap(t *testing.T) {
	min := 10
	bm1 := course.CreateBitMap(min, []int{15, 30, 20, 50, 23})
	bm2 := course.CreateBitMap(min, []int{30, 15, 50, 20, 23, 45})
	fmt.Println(course.IntersectionOfBitMap(bm1, bm2, min))
}

func TestIntersectionOfOrderedList(t *testing.T) {
	arr := []int{15, 20, 23, 30, 50}
	brr := []int{12, 15, 23, 30, 45, 50}
	fmt.Println(course.IntersectionOfOrderedList(arr, brr))
}

// go test -v ./course/test -run=^TestBits$ -count=1
// go test ./course/test -bench=^BenchmarkBitFilter -run=^$ -count=1 -benchmem
// go test -v ./course/test -run=^TestBitMap$ -count=1
// go test -v ./course/test -run=^TestIntersectionOfOrderedList$ -count=1
