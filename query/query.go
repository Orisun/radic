package query

import "strings"

type Query struct {
	TermQuery
	rangeQuerys []*RangeQuery
}

func NewQuery(termQuery TermQuery) *Query {
	if termQuery.Empty() {
		return nil
	}
	return &Query{
		TermQuery: termQuery,
	}
}

// Builder模式。方法返回结构体本身
func (q *Query) Filter(rangeQuery *RangeQuery) *Query {
	if !rangeQuery.Empty() {
		q.rangeQuerys = append(q.rangeQuerys, rangeQuery)
	}
	return q
}

func (q Query) Empty() bool {
	return q.TermQuery.Empty() && len(q.rangeQuerys) == 0
}

func (q Query) String() string {
	if q.Empty() {
		return ""
	}
	sb := strings.Builder{}
	if !q.TermQuery.Empty() {
		sb.WriteString("(")
		sb.WriteString(q.TermQuery.String())
		sb.WriteString(")")
		sb.WriteString(" and ")
	}
	for _, rg := range q.rangeQuerys {
		sb.WriteString("(")
		sb.WriteString(rg.String())
		sb.WriteString(")")
		sb.WriteString(" and ")
	}
	s := sb.String()
	s = s[:len(s)-5]
	return s
}
