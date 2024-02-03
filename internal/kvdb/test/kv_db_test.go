package test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Orisun/radic/v2/internal/kvdb"
)

var (
	db       kvdb.IKeyValueDB
	setup    func() //测试之前执行一些初始化工作
	teardown func() //测试之后执行一些收尾工作
)

func init() {
	teardown = func() {
		db.Close()
	}
}

func testGetDbPath(db kvdb.IKeyValueDB) error {
	fmt.Println("db path", db.GetDbPath())
	return nil
}

func testGetSetDelete(db kvdb.IKeyValueDB) error {
	k1 := []byte("k1")
	v1 := []byte("v1")
	k2 := []byte("k2")
	v2 := []byte("v2")
	// 写入<k, v>
	db.Set(k1, v1)
	db.Set(k2, v2)
	// 读取<k, v>
	v, err := db.Get(k1)
	if err != nil {
		return err
	}
	fmt.Println("v1=", string(v))
	v, err = db.Get(k2)
	if err != nil {
		return err
	}
	fmt.Println("v2=", string(v))

	// 删除<k, v>
	err = db.Delete(k1)
	if err != nil {
		return err
	}
	err = db.Delete(k2)
	if err != nil {
		return err
	}

	// 读取<k, v>
	_, err = db.Get(k1)
	if err == nil {
		return errors.New("key已被删除，却仍能读出数据")
	}
	_, err = db.Get(k2)
	if err == nil {
		return errors.New("key已被删除，却仍能读出数据")
	}

	//判断key是否存在
	fmt.Printf("k1存在 %t\n", db.Has(k1))
	fmt.Printf("k2存在 %t\n", db.Has(k2))

	return nil
}

func testBatchGetSetDelete(db kvdb.IKeyValueDB) error {
	k1 := []byte("k1")
	v1 := []byte("v1")
	k2 := []byte("k2")
	v2 := []byte("v2")
	//批量写入<k, v>
	db.BatchSet([][]byte{k1, k2}, [][]byte{v1, v2})
	//批量读取
	values, err := db.BatchGet([][]byte{k1, k2})
	if err != nil {
		return err
	}
	fmt.Printf("values ")
	for _, v := range values {
		fmt.Printf("%s ", string(v))
	}
	fmt.Println()

	//批量删除
	db.BatchDelete([][]byte{k1, k2})
	//批量读取
	// 读取<k, v>
	_, err = db.Get(k1)
	if err == nil {
		return errors.New("key已被删除，却仍能读出数据")
	}
	_, err = db.Get(k2)
	if err == nil {
		return errors.New("key已被删除，却仍能读出数据")
	}
	fmt.Println()

	//判断key是否存在
	fmt.Printf("k1存在 %t\n", db.Has(k1))
	fmt.Printf("k2存在 %t\n", db.Has(k2))

	return nil
}

func testIterDB(db kvdb.IKeyValueDB) error {
	k1 := []byte("k1")
	v1 := []byte("v1")
	k2 := []byte("k2")
	v2 := []byte("v2")
	//批量写入<k, v>
	db.BatchSet([][]byte{k1, k2}, [][]byte{v1, v2})

	fmt.Println("遍历DB")
	db.IterDB(func(k, v []byte) error {
		fmt.Printf("key=%s value=%s\n", string(k), string(v))
		return nil
	})
	fmt.Println("遍历key")
	db.IterKey(func(k []byte) error {
		fmt.Printf("key=%s\n", string(k))
		return nil
	})

	return nil
}

func testPipeline(t *testing.T) { //整个测试流
	defer teardown()
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
}
