package test

import (
	"fmt"
	"testing"

	"github.com/Orisun/radic/v2/query"
)

func TestQuery(t *testing.T) {
	rangeQ1 := query.NewRangeQuery(FIELD).WithGte(100).WithLte(200)
	rangeQ2 := query.NewRangeQuery("height").WithGte(1.5).WithLte(1.8)
	termQ := query.NewTermQuery(FIELD, "java")
	q := query.NewQuery(termQ).Filter(rangeQ1).Filter(rangeQ2)
	fmt.Println(q)
}

// go test -v ./query/test -run=^TestQuery$ -count=1
