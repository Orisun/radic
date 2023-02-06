// Date         : 2018/7/28
// Author       : zhangchaoyang
// Description  :
package types

import (
	"runtime"
)

const (
	defaultIndexName             = "radic"
	defaultAddCacheSize          = 10000
	defaultDeleteCacheSize       = 1000
	defaultInvertIndexSegmentNum = 32
	defaultForwardIndexSize      = 100000
	defaultDbShardNum            = 16
	defaultDbPath                = "dbs"
	defaultDbType                = "badger"
	defaultMaxRetrieve           = 5000
	defaultMaxInvertListLen      = 50000
)

var (
	defaultAddDocThreads    = runtime.NumCPU()
	defaultDeleteDocThreads = runtime.NumCPU() / 2
)

type IndexerOpts struct {
	IndexName             string //索引名
	AddCacheSize          int    //add任务channel的容量
	DeleteCacheSize       int    //delete任务channel的容量
	AddDocThreads         int    //add任务开多少个并了
	DeleteDocThreads      int    //delete任务开多少个并了
	InvertIndexSegmentNum int    //倒排索引被划分成几个map，减少并发读写同一个Map的冲突
	ForwardIndexSize      int    //正排索引在内存中缓存多少条记录。其余的交给localDB(badger等)
	DbShardNum            int    //启用多少个localDB(实际上启用了2*DbShardNum个localDB，因为key和detail占了两份)
	DbPath                string //localDB的本地目录
	DbType                string //可以使用"badger"、"rocksdb"和"bolt"，rocksdb要求服务器上安装很多rocksDB相关的组件
	MaxRetrieve           int    //最多只返回多少条结果。超过MaxRetrieve时会执行截断，截断前会先按照RankScore从大到小排序
	MaxInvertListLen      int    //每条倒排链最多只读取前MaxInvertListLen个docid
}

func (options *IndexerOpts) Init() {
	if options.IndexName == "" {
		options.IndexName = defaultIndexName
	}
	if options.AddCacheSize == 0 {
		options.AddCacheSize = defaultAddCacheSize
	}
	if options.DeleteCacheSize == 0 {
		options.DeleteCacheSize = defaultDeleteCacheSize
	}
	if options.InvertIndexSegmentNum == 0 {
		options.InvertIndexSegmentNum = defaultInvertIndexSegmentNum
	}
	if options.ForwardIndexSize == 0 {
		options.ForwardIndexSize = defaultForwardIndexSize
	}
	if options.DbShardNum == 0 {
		options.DbShardNum = defaultDbShardNum
	}
	if options.DbPath == "" {
		options.DbPath = defaultDbPath
	}
	if options.DbType == "" {
		options.DbType = defaultDbType
	}
	if options.AddDocThreads == 0 {
		options.AddDocThreads = defaultAddDocThreads
	}
	if options.DeleteDocThreads == 0 {
		options.DeleteDocThreads = defaultDeleteDocThreads
	}
	if options.MaxRetrieve == 0 {
		options.MaxRetrieve = defaultMaxRetrieve
	}
	if options.MaxInvertListLen == 0 {
		options.MaxInvertListLen = defaultMaxInvertListLen
	}
}
