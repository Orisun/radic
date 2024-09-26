package test

import (
	"testing"

	"github.com/Orisun/radic/v2/internal/kvdb"
	"github.com/Orisun/radic/v2/types"
	"github.com/Orisun/radic/v2/util"
)

func TestBolt(t *testing.T) {
	setup = func() {
		var err error
		db, err = kvdb.GetKvDb(types.BOLT, util.RootPath+"data/bolt_db") //使用工厂模式
		if err != nil {
			panic(err)
		}
	}

	t.Run("bolt_test", testPipeline)
}

// go test -v ./internal/kvdb/test -run=^TestBolt$ -count=1
