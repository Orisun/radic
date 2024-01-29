package test

import (
	"fmt"
	"testing"

	"github.com/Orisun/radic/v2/types"
)

const FIELD = ""

// ((A|B|C)&D)|E&((F|G)&H)

func TestTermQuery(t *testing.T) {
	A := types.NewTermQuery(FIELD, "") //空Expression
	B := types.NewTermQuery(FIELD, "B")
	C := types.NewTermQuery(FIELD, "C")
	D := types.NewTermQuery(FIELD, "D")
	E := &types.TermQuery{} //空Expression
	F := types.NewTermQuery(FIELD, "F")
	G := types.NewTermQuery(FIELD, "G")
	H := types.NewTermQuery(FIELD, "H")

	var q *types.TermQuery

	q = A
	fmt.Println(q.ToString()) //print函数会自动调用变量的String()方法

	q = B.Or(C)
	fmt.Println(q.ToString())

	// ((A|B|C)&D)|E&((F|G)&H)
	q = A.Or(B).Or(C).And(D).Or(E).And(F.Or(G)).And(H)
	fmt.Println(q.ToString())
}

// go test -v ./types/test -run=^TestTermQuery$ -count=1
