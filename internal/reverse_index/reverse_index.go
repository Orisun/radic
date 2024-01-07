package reverseindex

import "github.com/Orisun/radic/v2/types"

type IReverseIndexer interface {
	Add(DocId uint32, FeatureBits uint32, Keywords []string)                                //添加一个doc
	Delete(DocId uint32, Keyword string)                                                    //从key上删除对应的doc
	Search(query types.TermQuery, onFlag uint32, offFlag uint32, orFlags []uint32) []uint32 //查找
	Iter(func(key string, value any))                                                       //遍历倒排索引（用于调试）
}
