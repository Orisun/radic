package test

import (
	"fmt"
	"testing"

	"github.com/Orisun/radic/v2/query"
)

func TestRangeQuery(t *testing.T) {
	q := query.NewRangeQuery(FIELD)
	if !q.Empty() {
		t.Fail()
	}

	q = query.NewRangeQuery(FIELD).WithGte(100)
	fmt.Println(q)

	q = query.NewRangeQuery(FIELD).WithLte(200)
	fmt.Println(q)

	q = query.NewRangeQuery(FIELD).WithGte(100).WithLte(200)
	fmt.Println(q)
}

// go test -v ./query/test -run=^TestRangeQuery$ -count=1
