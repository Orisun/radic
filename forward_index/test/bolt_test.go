// Date         : 2018/7/30 下午12:00
// Author       : zhangchaoyang
// Description  :
package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	storage "github.com/Orisun/radic/forward_index"
	"github.com/Orisun/radic/types"
	"os"
	"testing"
)

var (
	boltDB storage.Storage
)

func Init3() {
	var err error
	if boltDB, err = storage.OpenStorage("bolt", "dbs"); err == nil {
		if boltDB == nil {
			fmt.Println("open storage failed")
			os.Exit(1)
		}
	} else {
		panic(err)
	}
	//gob.Register(Talent{})
}

func TestBoltGetSet(t *testing.T) {
	t.SkipNow()
	Init3()
	defer boltDB.Close()
	//gob.Register(Talent{})
	talent1 := Talent{
		Id:    111111,
		Age:   28,
		Name:  "orisun",
		Is211: true,
	}
	talent2 := Talent{
		Id:    222222,
		Age:   30,
		Name:  "zcy",
		Is211: true,
	}
	kw1 := &types.Keyword{Field: "company", Word: "百度"}
	kw2 := &types.Keyword{Field: "skill", Word: "测试"}
	kw3 := &types.Keyword{Field: "name", Word: talent1.Name}
	kw4 := &types.Keyword{Field: "skill", Word: "java"}
	kw5 := &types.Keyword{Field: "skill", Word: "机器学习"}
	kw6 := &types.Keyword{Field: "name", Word: talent2.Name}
	docInfo1 := types.DocInfo{
		DocId:     uint32(talent1.Id),
		Keyword:   []*types.Keyword{kw1, kw2, kw3},
		RankScore: 2,
		Entry:     talent1.Serialize(),
	}
	docInfo2 := types.DocInfo{
		DocId:     uint32(talent2.Id),
		Keyword:   []*types.Keyword{kw1, kw4, kw5, kw6},
		RankScore: 3,
		Entry:     talent2.Serialize(),
	}

	key1 := getStorageKey(docInfo1.DocId)
	var value1 bytes.Buffer
	encoder := gob.NewEncoder(&value1)
	if err := encoder.Encode(docInfo1); err == nil {
		boltDB.Set(key1, value1.Bytes())
	} else {
		fmt.Printf("write doc %d to boltDB failed\n", docInfo1.DocId)
		t.Fail()
	}

	key2 := getStorageKey(docInfo2.DocId)
	var value2 bytes.Buffer
	encoder = gob.NewEncoder(&value2)
	if err := encoder.Encode(docInfo2); err == nil {
		boltDB.Set(key2, value2.Bytes())
	} else {
		fmt.Printf("write doc %d to boltDB failed\n", docInfo2.DocId)
		t.Fail()
	}

	if datasBytes, error := boltDB.Get(key1); error == nil {
		buf := bytes.NewReader(datasBytes)
		dec := gob.NewDecoder(buf)
		var data types.DocInfo
		err := dec.Decode(&data)
		if err == nil {
			fmt.Println(data.DocId)
			fmt.Println(data.Keyword)
			fmt.Println(data.RankScore)
			talent := DeserializeTalent(data.Entry)
			fmt.Println(talent.Id)
			fmt.Println(talent.Name)
			fmt.Println(talent.Age)
			fmt.Println(talent.Is211)
		} else {
			t.Fail()
		}
	} else {
		t.Fail()
	}
	fmt.Println("=======================")

	fmt.Println("iterate boltDB")
	boltDB.IterDB(func(k, v []byte) error {
		buf := bytes.NewReader(v)
		dec := gob.NewDecoder(buf)
		var data types.DocInfo
		err := dec.Decode(&data)
		if err == nil {
			fmt.Println(data.DocId)
			fmt.Println(data.Keyword)
			fmt.Println(data.RankScore)
			talent := DeserializeTalent(data.Entry)
			fmt.Println(talent.Id)
			fmt.Println(talent.Name)
			fmt.Println(talent.Age)
			fmt.Println(talent.Is211)
			fmt.Println("--------------------")
			return nil
		} else {
			t.Fail()
			return err
		}
	})
	fmt.Println("=======================")

	if !boltDB.Has(key1) {
		t.Fail()
	}
	if !boltDB.Has(key2) {
		t.Fail()
	}
	boltDB.Delete(key1)
	boltDB.Delete(key2)
	if boltDB.Has(key1) {
		t.Fail()
	}
	if boltDB.Has(key2) {
		t.Fail()
	}

	boltDB.BatchSet([][]byte{key1, key2}, [][]byte{value1.Bytes(), value2.Bytes()})
	values, _ := boltDB.BatchGet([][]byte{key1, key2})
	for _, value := range values {
		buf := bytes.NewReader(value)
		dec := gob.NewDecoder(buf)
		var data types.DocInfo
		err := dec.Decode(&data)
		if err == nil {
			fmt.Println(data.DocId)
			fmt.Println(data.Keyword)
			fmt.Println(data.RankScore)
			talent := DeserializeTalent(data.Entry)
			fmt.Println(talent.Id)
			fmt.Println(talent.Name)
			fmt.Println(talent.Age)
			fmt.Println(talent.Is211)
			fmt.Println("--------------------")
		} else {
			t.Fail()
		}
	}
	fmt.Println("=======================")

	boltDB.BatchDelete([][]byte{key1, key2})
	if boltDB.Has(key1) {
		t.Fail()
	}
	if boltDB.Has(key2) {
		t.Fail()
	}
}
