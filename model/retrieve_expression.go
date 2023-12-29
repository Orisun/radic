// Deprecated 这是反面教材，多层函数嵌套。正面教材参见 package query
package model

import "strings"

type RetrieveExpression struct {
	Must    []RetrieveExpression
	Should  []RetrieveExpression
	Keyword string
}

func (exp RetrieveExpression) Empty() bool {
	return len(exp.Keyword) == 0 && len(exp.Must) == 0 && len(exp.Should) == 0
}

func KeywordExpression(keyword string) RetrieveExpression {
	return RetrieveExpression{Keyword: keyword}
}

func MustExpression(exps ...RetrieveExpression) RetrieveExpression {
	if len(exps) == 0 {
		return RetrieveExpression{}
	}
	array := make([]RetrieveExpression, 0, len(exps))
	//非空的Expression才能添加到array里面去
	for _, exp := range exps {
		if !exp.Empty() {
			array = append(array, exp)
		}
	}
	return RetrieveExpression{Must: array}
}

func ShouldExpression(exps ...RetrieveExpression) RetrieveExpression {
	if len(exps) == 0 {
		return RetrieveExpression{}
	}
	array := make([]RetrieveExpression, 0, len(exps))
	//非空的Expression才能添加到array里面去
	for _, exp := range exps {
		if !exp.Empty() {
			array = append(array, exp)
		}
	}
	return RetrieveExpression{Should: array}
}

//print函数会自动调用变量的String()方法
func (exp RetrieveExpression) String() string {
	if len(exp.Keyword) > 0 {
		return exp.Keyword
	} else if len(exp.Must) > 0 {
		if len(exp.Must) == 1 {
			return exp.Must[0].String()
		} else {
			sb := strings.Builder{}
			sb.WriteByte('(')
			for _, e := range exp.Must {
				sb.WriteString(e.String())
				sb.WriteByte('&')
			}
			s := sb.String()
			s = s[0:len(s)-1] + ")"
			return s
		}
	} else if len(exp.Should) > 0 {
		if len(exp.Should) == 1 {
			return exp.Should[0].String()
		} else {
			sb := strings.Builder{}
			sb.WriteByte('(')
			for _, e := range exp.Should {
				sb.WriteString(e.String())
				sb.WriteByte('|')
			}
			s := sb.String()
			s = s[0:len(s)-1] + ")"
			return s
		}

	}
	return ""
}
