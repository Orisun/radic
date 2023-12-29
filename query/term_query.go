package query

import "strings"

type TermQuery struct {
	Field   string
	Keyword string
	Must    []TermQuery
	Should  []TermQuery
}

func NewTermQuery(field, keyword string) TermQuery {
	return TermQuery{Field: field, Keyword: keyword}
}

func (q TermQuery) Empty() bool {
	return (len(q.Field) == 0 || len(q.Keyword) == 0) && len(q.Must) == 0 && len(q.Should) == 0
}

// Builder模式。方法返回结构体本身
func (q TermQuery) And(querys ...TermQuery) TermQuery {
	if len(querys) == 0 {
		return q
	}
	array := make([]TermQuery, 0, 1+len(querys))
	//空的query会被排除掉
	if !q.Empty() {
		array = append(array, q)
	}
	for _, ele := range querys {
		if !ele.Empty() {
			array = append(array, ele)
		}
	}
	return TermQuery{Must: array}
}

func (q TermQuery) Or(querys ...TermQuery) TermQuery {
	if len(querys) == 0 {
		return q
	}
	array := make([]TermQuery, 0, 1+len(querys))
	//空的query会被排除掉
	if !q.Empty() {
		array = append(array, q)
	}
	for _, ele := range querys {
		if !ele.Empty() {
			array = append(array, ele)
		}
	}
	return TermQuery{Should: array}
}

//print函数会自动调用变量的String()方法
func (q TermQuery) String() string {
	if len(q.Keyword) > 0 {
		return q.Field + "_" + q.Keyword
	} else if len(q.Must) > 0 {
		if len(q.Must) == 1 {
			return q.Must[0].String()
		} else {
			sb := strings.Builder{}
			sb.WriteByte('(')
			for _, e := range q.Must {
				sb.WriteString(e.String())
				sb.WriteByte('&')
			}
			s := sb.String()
			s = s[0:len(s)-1] + ")"
			return s
		}
	} else if len(q.Should) > 0 {
		if len(q.Should) == 1 {
			return q.Should[0].String()
		} else {
			sb := strings.Builder{}
			sb.WriteByte('(')
			for _, e := range q.Should {
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
