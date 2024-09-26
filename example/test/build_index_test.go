package test

import (
	"os"
	"testing"

	demo "github.com/Orisun/radic/v2/example"
	"github.com/Orisun/radic/v2/index_service"
	"github.com/Orisun/radic/v2/types"
	"github.com/Orisun/radic/v2/util"
)

var (
	dbType  = types.BOLT
	dbPath  = util.RootPath + "data/local_db/video_bolt"
	indexer *index_service.Indexer
)

func Init() {
	os.Remove(dbPath) //先删除原有的索引文件
	indexer = new(index_service.Indexer)
	if err := indexer.Init(50000, dbType, dbPath); err != nil {
		panic(err)
	}
}

func TestBuildIndexFromFile(t *testing.T) {
	Init()
	defer indexer.Close()
	csvFile := util.RootPath + "data/bili_video.csv"
	demo.BuildIndexFromFile(csvFile, indexer, 0, 0)
}

// go test -v ./demo/test -run=^TestBuildIndexFromFile$ -count=1
