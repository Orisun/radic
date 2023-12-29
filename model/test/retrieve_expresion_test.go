package test

import (
	"fmt"
	"testing"

	"github.com/Orisun/radic/v2/model"
)

func TestRetrieveExpression(t *testing.T) {
	// ((A|B|C)&D)|E&((F|G)&H)
	A := model.KeywordExpression("") //空Expression
	B := model.KeywordExpression("B")
	C := model.KeywordExpression("C")
	D := model.KeywordExpression("D")
	E := model.RetrieveExpression{} //空Expression
	F := model.KeywordExpression("F")
	G := model.KeywordExpression("G")
	H := model.KeywordExpression("H")

	var exp model.RetrieveExpression

	exp = A
	fmt.Println(exp) //print函数会自动调用变量的String()方法

	exp = model.ShouldExpression(A, B, C)
	fmt.Println(exp)

	//函数嵌套的导数太多，编码时需要非常小心
	exp = model.MustExpression(model.ShouldExpression(model.MustExpression(model.ShouldExpression(A, B, C), D), E), model.MustExpression(model.ShouldExpression(F, G), H))
	fmt.Println(exp)
}

// go test -v ./model/test -run=^TestRetrieveExpression$ -count=1
