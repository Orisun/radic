// Date         : 2018/7/30 下午12:00
// Author       : zhangchaoyang
// Description  :
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	storage "github.com/Orisun/radic/forward_index"
	"github.com/Orisun/radic/types"
	"log"
	"os"
	"testing"
)

var (
	badgerDB storage.Storage
)

func Init2() {
	var err error
	if badgerDB, err = storage.OpenStorage("badger", "dbs"); err == nil {
		if badgerDB == nil {
			log.Printf("open storage failed")
			os.Exit(1)
		}
	} else {
		panic(err)
	}
	//gob.Register(Talent{})
}

type Talent struct {
	Id    uint32
	Age   int
	Name  string
	Is211 bool
}

func (self *Talent) Serialize() []byte {
	var value bytes.Buffer
	encoder := gob.NewEncoder(&value) //gob是go自带的序列化方法
	err := encoder.Encode(self)
	if err == nil {
		return value.Bytes()
	} else {
		fmt.Println(err)
		return []byte{}
	}
}

func DeserializeTalent(v []byte) *Talent {
	buf := bytes.NewReader(v)
	dec := gob.NewDecoder(buf)
	var data = Talent{}
	err := dec.Decode(&data)
	if err == nil {
		return &data
	} else {
		fmt.Println(err)
		return nil
	}
}

func TestBadgerGetSet(t *testing.T) {
	t.SkipNow()
	Init2()
	defer badgerDB.Close()
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
		badgerDB.Set(key1, value1.Bytes())
	} else {
		fmt.Printf("write doc %d to badgerDB failed\n", docInfo1.DocId)
		t.Fail()
	}

	key2 := getStorageKey(docInfo2.DocId)
	var value2 bytes.Buffer
	encoder = gob.NewEncoder(&value2)
	if err := encoder.Encode(docInfo2); err == nil {
		badgerDB.Set(key2, value2.Bytes())
	} else {
		fmt.Printf("write doc %d to badgerDB failed\n", docInfo2.DocId)
		t.Fail()
	}

	if datasBytes, error := badgerDB.Get(key1); error == nil {
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

	fmt.Println("iterate badgerDB")
	badgerDB.IterDB(func(k, v []byte) error {
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

	if !badgerDB.Has(key1) {
		t.Fail()
	}
	if !badgerDB.Has(key2) {
		t.Fail()
	}
	badgerDB.Delete(key1)
	badgerDB.Delete(key2)
	if badgerDB.Has(key1) {
		t.Fail()
	}
	if badgerDB.Has(key2) {
		t.Fail()
	}

	badgerDB.BatchSet([][]byte{key1, key2}, [][]byte{value1.Bytes(), value2.Bytes()})
	values, _ := badgerDB.BatchGet([][]byte{key1, key2})
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

	badgerDB.BatchDelete([][]byte{key1, key2})
	if badgerDB.Has(key1) {
		t.Fail()
	}
	if badgerDB.Has(key2) {
		t.Fail()
	}
}

//getStorageKey
func getStorageKey(docId uint32) []byte {
	key := make([]byte, 8)
	cnt := binary.PutUvarint(key, uint64(docId))
	return key[:cnt]
}
