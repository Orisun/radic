package query

import (
	"fmt"
	"strings"
)

type RangeQuery struct {
	Field      string
	Gte        float64
	includeGte bool
	Lte        float64
	includeLte bool
}

func NewRangeQuery(field string) *RangeQuery {
	return &RangeQuery{Field: field}
}

// Builder模式。方法返回结构体本身
func (q *RangeQuery) WithGte(arg float64) *RangeQuery {
	q.includeGte = true
	q.Gte = arg
	return q
}

func (q *RangeQuery) WithLte(arg float64) *RangeQuery {
	q.includeLte = true
	q.Lte = arg
	return q
}

func (q RangeQuery) Empty() bool {
	return len(q.Field) == 0 || (!q.includeGte && !q.includeLte)
}

func (q RangeQuery) String() string {
	if q.Empty() {
		return ""
	}
	sb := strings.Builder{}
	if q.includeGte {
		sb.WriteString(q.Field + ">=" + fmt.Sprintf("%g", q.Gte) + " and ")
	}
	if q.includeLte {
		sb.WriteString(q.Field + "<=" + fmt.Sprintf("%g", q.Lte) + " and ")
	}
	s := sb.String()
	s = s[:len(s)-5]
	return s
}
