package test

import (
	"os"
	"testing"

	"github.com/Orisun/radic/v2"
	"github.com/Orisun/radic/v2/demo"
	"github.com/Orisun/radic/v2/internal/kvdb"
	"github.com/Orisun/radic/v2/util"
)

var (
	dbType  = kvdb.BOLT
	dbPath  = util.RootPath + "data/local_db/vedio_bolt"
	indexer *radic.Indexer
)

func Init() {
	os.Remove(dbPath) //先删除原有的索引文件
	indexer = new(radic.Indexer)
	if err := indexer.Init(50000, dbType, dbPath); err != nil {
		panic(err)
	}
}

func TestBuildIndexFromFile(t *testing.T) {
	Init()
	defer indexer.Close()
	csvFile := util.RootPath + "data/bili_video.csv"
	demo.BuildIndexFromFile(csvFile, indexer)
}

// go test -v ./demo/test -run=^TestBuildIndexFromFile$ -count=1
