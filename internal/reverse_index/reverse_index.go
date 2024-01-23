package reverseindex

import "github.com/Orisun/radic/v2/types"

type IReverseIndexer interface {
	Add(doc types.Document)                                                              //添加一个doc
	Delete(IntId uint64, keyword *types.Keyword)                                         //从key上删除对应的doc
	Search(q *types.TermQuery, onFlag uint64, offFlag uint64, orFlags []uint64) []string //查找,返回业务侧文档ID
}
