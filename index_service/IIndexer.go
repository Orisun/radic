package index_service

import types "github.com/Orisun/radic/v2/types"

//Sentinel（分布式grpc的哨兵）和Indexer（单机索引）都实现了该接口
type IIndexer interface {
	AddDoc(doc types.Document) (int, error)
	DeleteDoc(docId string) int
	Search(query *types.TermQuery, onFlag uint64, offFlag uint64, orFlags []uint64) []*types.Document
	Count() int
	Close() error
}
