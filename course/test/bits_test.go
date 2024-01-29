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
// go test -v ./course/test -run=^TestBitMap$ -count=1
// go test -v ./course/test -run=^TestIntersectionOfOrderedList$ -count=1
