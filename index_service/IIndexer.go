package index_service

import types "github.com/Orisun/radic/v2/types"

type IIndexer interface {
	AddDoc(doc types.Document) (int, error)
	DeleteDoc(docId string) int
	Search(query *types.TermQuery, onFlag uint64, offFlag uint64, orFlags []uint64) []*types.Document
}
