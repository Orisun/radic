package test

import (
	"testing"

	"github.com/Orisun/radic/v2/internal/kvdb"
	"github.com/Orisun/radic/v2/types"
	"github.com/Orisun/radic/v2/util"
)

func TestBadger(t *testing.T) {
	setup = func() {
		var err error
		db, err = kvdb.GetKvDb(types.BADGER, util.RootPath+"data/badger_db")
		if err != nil {
			panic(err)
		}
	}

	t.Run("badger_test", testPipeline)
}

// go test -v ./internal/kvdb/test -run=^TestBadger$ -count=1
