package test

import (
	"fmt"
	"testing"

	"github.com/Orisun/radic/v2/types"
)

func TestRetrieveExpression(t *testing.T) {
	// ((A|B|C)&D)|E&((F|G)&H)
	A := types.KeywordExpression("") //空Expression
	B := types.KeywordExpression("B")
	C := types.KeywordExpression("C")
	D := types.KeywordExpression("D")
	E := types.RetrieveExpression{} //空Expression
	F := types.KeywordExpression("F")
	G := types.KeywordExpression("G")
	H := types.KeywordExpression("H")

	var exp types.RetrieveExpression

	exp = A
	fmt.Println(exp) //print函数会自动调用变量的String()方法

	exp = types.ShouldExpression(A, B, C)
	fmt.Println(exp)

	//函数嵌套的导数太多，编码时需要非常小心
	exp = types.MustExpression(types.ShouldExpression(types.MustExpression(types.ShouldExpression(A, B, C), D), E), types.MustExpression(types.ShouldExpression(F, G), H))
	fmt.Println(exp)
}

// go test -v ./types/test -run=^TestRetrieveExpression$ -count=1
