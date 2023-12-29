package test

import (
	"fmt"
	"testing"

	"github.com/Orisun/radic/v2/query"
)

const FIELD = "title"

func TestTermQuery(t *testing.T) {
	// A|B|C&D|E&(F|G)&H
	A := query.NewTermQuery(FIELD, "") //空Expression
	B := query.NewTermQuery(FIELD, "B")
	C := query.NewTermQuery(FIELD, "C")
	D := query.NewTermQuery(FIELD, "D")
	E := query.TermQuery{} //空Expression
	F := query.NewTermQuery(FIELD, "F")
	G := query.NewTermQuery(FIELD, "G")
	H := query.NewTermQuery(FIELD, "H")

	var q query.TermQuery

	q = A
	fmt.Println(q) //print函数会自动调用变量的String()方法

	q = B.Or(C)
	fmt.Println(q)

	q = A.Or(B).Or(C).And(D).Or(E).And(F.Or(G)).And(H)
	fmt.Println(q)
}

// go test -v ./query/test -run=^TestTermQuery$ -count=1
