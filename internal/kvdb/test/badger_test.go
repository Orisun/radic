package test

import (
	"fmt"
	"testing"

	"github.com/Orisun/radic/v2/internal/kvdb"
	"github.com/Orisun/radic/v2/util"
)

func TestBadger(t *testing.T) {
	var db kvdb.IKeyValueDB
	setup := func() {
		var err error
		db, err = kvdb.GetKvDb(kvdb.BADGER, util.RootPath+"data/badger_db")
		if err != nil {
			panic(err)
		}
	}

	t.Run("badger_test", func(t *testing.T) {
		defer func() {
			db.Close()
		}()
		setup()

		testGetDbPath(db)
		fmt.Println()

		err := testGetSetDelete(db)
		if err != nil {
			fmt.Println(err)
			t.Fail()
		}
		fmt.Println()

		err = testBatchGetSetDelete(db)
		if err != nil {
			fmt.Println(err)
			t.Fail()
		}
		fmt.Println()

		err = testIterDB(db)
		if err != nil {
			fmt.Println(err)
			t.Fail()
		}
		fmt.Println()
	})
}

// go test -v ./internal/kvdb/test -run=^TestBadger$ -count=1
