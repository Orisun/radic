// Date         : 2018/7/29
// Author       : zhangchaoyang
// Description  :
package radic

import (
	"context"
	"time"

	core "github.com/Orisun/radic/inverted_index"
	"github.com/Orisun/radic/types"
)

type IndexEngine struct {
	indexer core.Indexer
}

// Init 初始化参数，并启动引擎的各个后台协程
func (engine *IndexEngine) Start(indexOpts types.IndexerOpts) {
	engine.indexer.Init(indexOpts)
}

// Destroy 关闭引擎，释放各种资源
func (engine *IndexEngine) Stop() {
	engine.indexer.Close()
}

// IndexDoc 往索引上添加一个doc。异步执行的，不会立即生效
func (engine *IndexEngine) IndexDoc(doc types.DocInfo) {
	engine.indexer.IndexDoc(doc)
}

// BatchIndexDoc 往索引上添加一批doc。同步执行
func (engine *IndexEngine) BatchIndexDoc(docs []*types.DocInfo) {
	engine.indexer.BatchIndexDoc(docs)
}

// RemoveDoc 从索引上删除doc。异步执行的，不会立即生效
func (engine *IndexEngine) RemoveDoc(docId uint32) {
	engine.indexer.RemoveDoc(docId)
}

// Search 根据条件去索引上搜索doc
func (engine *IndexEngine) Search(request types.SearchRequest) types.SearchResp {
	must := request.Must
	should := request.Should
	not := request.Not
	orderLess := request.Orderless
	offset := request.OutputOffset
	var timeout int32 = 2000 //默认超时2秒
	if request.Timeout > 0 {
		timeout = request.Timeout //以request.Timeout为准
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
	defer cancel()
	result := make(chan []*types.DocInfo, 1)
	go func() {
		result <- engine.indexer.Lookup(must, should, not, orderLess, request.FilterIds, request.OnFlag, request.OffFlag, request.OrFlags)
	}()

	resp := types.SearchResp{}
	select {
	case docInfos := <-result:
		resp.Total = int32(len(docInfos))
		if !request.CountDocsOnly && resp.Total > offset {
			count := resp.Total - offset
			docs := make([][]byte, count)
			for i := offset; i < offset+count; i++ {
				docs[i] = docInfos[i].Entry
			}
			resp.Docs = docs
		}
	case <-ctx.Done():
		resp.Timeout = true
	}

	return resp
}

// IterIndex 遍历索引里的数据
func (engine *IndexEngine) IterIndex(fn func(docId uint32, docInfo types.DocInfo) error) {
	engine.indexer.IterIndex(fn)
}

// IterKeyOnIndex 遍历索引里的key
func (engine *IndexEngine) IterKeyOnIndex(fn func(docId uint32) error) error {
	return engine.indexer.IterKeyOnIndex(fn)
}

// GetDoc 根据ID获取一个doc的详情。flag=1仅获取detail，flag=0都获取
func (engine *IndexEngine) GetDoc(docId uint32, flag int) *types.DocInfo {
	return engine.indexer.GetDoc(docId, flag)
}

// GetDocs 根据ID获取一批doc的详情
func (engine *IndexEngine) GetDocs(docIds []uint32) []*types.DocInfo {
	return engine.indexer.GetDocs(docIds)
}

// LoadDataFromDb2InvertIndex 从本地db加载数据，加到倒排索引中。仅支持在系统启动时一次性调用
func (engine *IndexEngine) LoadDataFromDb2InvertIndex() int {
	return engine.indexer.LoadDataFromDb2InvertIndex()
}

// QuickLoadDataFromDb2InvertIndex 从本地db加载数据，加到倒排索引中。仅支持在系统启动时一次性调用
func (engine *IndexEngine) QuickLoadDataFromDb2InvertIndex(concurrency int) int {
	return engine.indexer.QuickLoadDataFromDb2InvertIndex(concurrency)
}

// DocNum 索引上的文档总数
func (engine *IndexEngine) DocNum() int {
	return engine.indexer.DocNum()
}

// DeleteFromInvertIndex 从倒排索引的一个key上把指定docID删掉。这是异步执行的，积累到一定量才会真正执行。彻底执行请调用FlushDeleteOnInvertedIndex
func (engine *IndexEngine) DeleteFromInvertIndex(keyword *types.Keyword, docID uint32) {
	engine.indexer.AddDeletePair(docID, keyword)
}

// AddToInvertIndex 向倒排上添加。立即执行
func (engine *IndexEngine) AddToInvertIndex(keywords []*types.Keyword, doc *types.DocInfo) {
	engine.indexer.AddToInvertIndex(keywords, doc)
}

// AddToLocalDB 向正排上添加。立即执行
func (engine *IndexEngine) AddToLocalDB(doc *types.DocInfo) {
	engine.indexer.AddToLocalDB(doc)
}

// FlushDeleteOnInvertedIndex 需要从倒排索引上删除的docID，彻底将其删除
func (engine *IndexEngine) FlushDeleteOnInvertedIndex() {
	engine.indexer.CleanToBeDeleteBuffer()
}

// GetLocalDbPath 获取local db的路径
func (engine *IndexEngine) GetLocalDbPath() string {
	return engine.indexer.GetDbPath()
}

// InvertIndexStatistics 统计倒排索引上每个关键词下面有几个docID
func (engine *IndexEngine) InvertIndexStatistics(outfile string) int {
	return engine.indexer.InvertIndexStatistics(outfile)
}

// GetInvertListLen 获取倒排链上有几个docid
func (engine *IndexEngine) GetInvertListLen(keyword *types.Keyword, onFlag uint32, offFlag uint32, orFlags []uint32) int32 {
	var count int32 = 0
	if keyword != nil && len(keyword.Word) > 0 {
		count = engine.indexer.GetInvertListLen(keyword, onFlag, offFlag, orFlags)
	}
	return count
}

//UpdateForwardIndex 更新正排数据，前提是doc的keys无变化
//func (engine *IndexEngine) UpdateForwardIndex(doc *types.DocInfo) {
//	engine.indexer.UpdateDetail(doc)
//}

/**
git checkout -b b14 rocksdbV1.4
git merge master
// 修改代码
git add .
git commit -m"加日志"
git tag -a rocksdbV1.5 -m"加日志"
git push origin rocksdbV1.5
*/
