package radic

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"

	"github.com/Orisun/radic/v2/internal/kvdb"
	reverseindex "github.com/Orisun/radic/v2/internal/reverse_index"
	"github.com/Orisun/radic/v2/types"
	"github.com/Orisun/radic/v2/util"
)

// 把正排和倒排索引放到一起
type Indexer struct {
	forwardIndex kvdb.IKeyValueDB
	reverseIndex reverseindex.IReverseIndexer
}

// 初始化索引
func (indexer *Indexer) Init(DocNumEstimate int, dbtype int, DataDir string) error {
	db, err := kvdb.GetKvDb(dbtype, DataDir)
	if err != nil {
		return err
	}
	indexer.forwardIndex = db
	indexer.reverseIndex = reverseindex.NewSkipListReverseIndex(DocNumEstimate)
	return nil
}

// 关闭索引
func (indexer *Indexer) Close() error {
	return indexer.forwardIndex.Close()
}

// 把int转为vint
//
// 返回的[]byte中不存在0
func Int2Varint(docId uint32) []byte {
	key := make([]byte, 8)
	cnt := binary.PutUvarint(key, uint64(docId))
	return key[:cnt]
}

// 向索引中添加文档
func (indexer *Indexer) AddDoc(doc types.DocInfo) {
	//doc.Entry写入正排索引
	indexer.forwardIndex.Set(Int2Varint(doc.DocId), doc.Entry)
	//doc.Keyword写入正排索引
	var value bytes.Buffer
	encoder := gob.NewEncoder(&value)
	if err := encoder.Encode(doc.Keyword); err == nil {
		indexer.forwardIndex.Set(append(Int2Varint(doc.DocId), 0), value.Bytes()) //key统一以0结尾，以此来区分value里存在的是doc.Entry还是doc.Keyword
	}

	//遍历每一个keyword，写入倒排索引
	Keywords := make([]string, 0, len(doc.Keyword))
	for _, keyword := range doc.Keyword {
		Keywords = append(Keywords, keyword.ToString())
	}
	indexer.reverseIndex.Add(doc.DocId, doc.CompositeFeature, Keywords)
}

// 从索引上删除文档
func (indexer *Indexer) DeleteDoc(docId uint32) {
	//先读正排索引，得到doc.Keyword
	keyBytes, err := indexer.forwardIndex.Get(append(Int2Varint(docId), 0))
	if err == nil {
		keysReader := bytes.NewReader([]byte{})
		if len(keyBytes) > 0 {
			keysReader.Reset(keyBytes)
			keysDecoder := gob.NewDecoder(keysReader)
			var keywords []*types.Keyword
			err3 := keysDecoder.Decode(&keywords)
			if err3 == nil {
				// 遍历每一个keyword，从倒排索引上删除
				for _, kw := range keywords {
					indexer.reverseIndex.Delete(docId, kw.ToString())
				}
			}
		}
	}
	//从正排上删除
	indexer.forwardIndex.Delete(Int2Varint(docId))
	indexer.forwardIndex.Delete(append(Int2Varint(docId), 0))
}

// 检索，返回文档列表
func (indexer *Indexer) Search(query types.TermQuery, onFlag uint32, offFlag uint32, orFlags []uint32) [][]byte {
	docIds := indexer.reverseIndex.Search(query, onFlag, offFlag, orFlags)
	if len(docIds) == 0 {
		return nil
	}
	keys := make([][]byte, 0, len(docIds))
	for _, docId := range docIds {
		keys = append(keys, Int2Varint(docId))
	}
	data, err := indexer.forwardIndex.BatchGet(keys)
	if err != nil {
		util.Log.Printf("read kvdb failed: %s", err)
		return nil
	}
	return data
}

// 遍历倒排索引
func (indexer *Indexer) IterReverseIndex(f func(key string, value any)) {
	indexer.reverseIndex.Iter(f)
}
