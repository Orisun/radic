// Date         : 2019-03-11 13:54
// Author       : zhangzhaoyang
// Description  :

package test

import (
	"fmt"
	"radic/types"
	"testing"
)

func TestTopScoreDoc(t *testing.T) {
	d0 := types.ScoredDoc{DocId: 0, RankScore: 8}
	d1 := types.ScoredDoc{DocId: 1, RankScore: 0}
	d2 := types.ScoredDoc{DocId: 2, RankScore: 8}
	d3 := types.ScoredDoc{DocId: 3, RankScore: 5}
	d4 := types.ScoredDoc{DocId: 4, RankScore: 9}
	d5 := types.ScoredDoc{DocId: 5, RankScore: 2}
	d6 := types.ScoredDoc{DocId: 6, RankScore: 1}
	d7 := types.ScoredDoc{DocId: 6, RankScore: 14}
	d8 := types.ScoredDoc{DocId: 6, RankScore: 34}
	d9 := types.ScoredDoc{DocId: 6, RankScore: 20}
	d10 := types.ScoredDoc{DocId: 6, RankScore: 9}

	arr := types.ScoredDocs{d0, d1, d2, d3, d4, d5, d6, d7, d8, d9, d10}
	k := 8
	arr.TopK(k)

	for i, ele := range arr {
		fmt.Printf("%d\t%f\n", i, ele.RankScore)
	}
}
