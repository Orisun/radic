package test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Orisun/radic/v2/types"
)

// ((A|B|C)&D)|E&((F|G)&H)

func should(s ...string) string {
	if len(s) == 0 {
		return ""
	}
	sb := strings.Builder{}
	sb.WriteString("(")
	for _, ele := range s {
		if len(ele) > 0 {
			sb.WriteString(ele + "|")
		}
	}
	rect := sb.String()
	return rect[0:len(rect)-1] + ")"
	// sb.WriteString(")")
	// return "(" + strings.Join(s, "|") + ")"
}

func must(s ...string) string {
	return "(" + strings.Join(s, "&") + ")"
}

func TestN(t *testing.T) {
	fmt.Println(must(should(must(should("A", "B", "C"), "D"), "E"), must(should("F", "G"), "H")))
}

func TestTermQueryV0(t *testing.T) {
	A := types.KeywordExpression("") //空Expression
	B := types.KeywordExpression("B")
	C := types.KeywordExpression("C")
	D := types.KeywordExpression("D")
	E := types.TermQueryV0{} //空Expression
	F := types.KeywordExpression("F")
	G := types.KeywordExpression("G")
	H := types.KeywordExpression("H")

	var exp types.TermQueryV0

	exp = A
	fmt.Println(exp) //print函数会自动调用变量的String()方法

	exp = types.ShouldExpression(A, B, C)
	fmt.Println(exp)

	// ((A|B|C)&D)|E&((F|G)&H)
	//函数嵌套的导数太多，编码时需要非常小心
	exp = types.MustExpression(types.ShouldExpression(types.MustExpression(types.ShouldExpression(A, B, C), D), E), types.MustExpression(types.ShouldExpression(F, G), H))
	fmt.Println(exp)
}

// go test -v ./types/test -run=^TestN$ -count=1
// go test -v ./types/test -run=^TestTermQueryV0$ -count=1
