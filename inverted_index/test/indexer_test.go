// Date         : 2018/7/30 下午2:23
// Author       : zhangchaoyang
// Description  :
package test

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"strconv"
	"testing"
	"time"

	core "github.com/Orisun/radic/inverted_index"
	"github.com/Orisun/radic/types"
)

var indexer core.Indexer

func init() {
	options := types.IndexerOpts{
		DbShardNum: 2,
	}
	indexer.Init(options)
	//docNum := indexer.LoadDataFromDb2InvertIndex()
	//fmt.Printf("load %d doc from local db\n", docNum)
}

type Talent struct {
	Id    int
	Age   int
	Name  string
	Is211 bool
}

func (talent *Talent) Serialize() []byte {
	var value bytes.Buffer
	encoder := gob.NewEncoder(&value) //gob是go自带的序列化方法
	err := encoder.Encode(talent)
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

func TestAddLookup(t *testing.T) {
	defer indexer.Close()
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
	docInfo1 := &types.DocInfo{
		DocId:            uint32(talent1.Id),
		Keyword:          []*types.Keyword{kw1, kw2, kw3},
		RankScore:        2,
		Entry:            talent1.Serialize(),
		CompositeFeature: 5,
	}
	docInfo2 := &types.DocInfo{
		DocId:            uint32(talent2.Id),
		Keyword:          []*types.Keyword{kw1, kw4, kw5, kw6},
		RankScore:        3,
		Entry:            talent2.Serialize(),
		CompositeFeature: 9,
	}
	var onFlag uint32 = 0
	var offFlag uint32 = 0
	onFlag |= 1 << 0
	offFlag |= 1 << 2
	orFlags := []uint32{}
	//indexer.IndexDoc(*docInfo1)
	//indexer.IndexDoc(*docInfo2)
	//time.Sleep(1000 * time.Millisecond) //IndexDoc是异步执行的，这里要稍等一会儿，等Add操作执行完成
	indexer.BatchIndexDoc([]*types.DocInfo{docInfo1, docInfo2}) //BatchIndexDoc是同步执行的
	//indexer.InvertIndexStatistics("") 建好索引后最好调用一次InvertIndexStatistics，有利于搜索速度的提升

	indexer.IterIndex(func(docId uint32, data types.DocInfo) error {
		fmt.Println(docId)
		fmt.Println(data.DocId)
		fmt.Println(data.Keyword)
		fmt.Println(data.RankScore)
		talent := DeserializeTalent(data.Entry)
		fmt.Println(talent.Id)
		fmt.Println(talent.Name)
		fmt.Println(talent.Age)
		fmt.Println(talent.Is211)
		fmt.Println("------------------")
		return nil
	})
	fmt.Println("===================")

	data := indexer.GetDoc(uint32(talent1.Id), 1)
	fmt.Println(data.DocId)
	fmt.Println(data.DocId)
	fmt.Println(data.RankScore)
	talent := DeserializeTalent(data.Entry)
	fmt.Println(talent.Id)
	fmt.Println(talent.Name)
	fmt.Println(talent.Age)
	fmt.Println(talent.Is211)
	fmt.Println("===================")

	datas := indexer.GetDocs([]uint32{uint32(talent1.Id), uint32(talent2.Id)})
	for _, data := range datas {
		fmt.Println(data.DocId)
		fmt.Println(data.DocId)
		fmt.Println(data.RankScore)
		talent := DeserializeTalent(data.Entry)
		fmt.Println(talent.Id)
		fmt.Println(talent.Name)
		fmt.Println(talent.Age)
		fmt.Println(talent.Is211)
		fmt.Println("------------------")
	}
	fmt.Println("===================")

	must := []*types.Keyword{kw1}
	should := []*types.Keyword{}
	not := []*types.Keyword{}
	filter := make(map[uint32]bool)
	results := indexer.Lookup(must, should, not, false, nil, onFlag, offFlag, orFlags)
	fmt.Println("有must无should，无序，搜索结果：") //talent1和talent2被召回，且talent1排在前面
	for _, doc := range results {
		fmt.Println(doc)
		t := DeserializeTalent(data.Entry)
		fmt.Println(t.Id)
		fmt.Println(t.Name)
		fmt.Println(t.Age)
		fmt.Println(t.Is211)
		fmt.Println("------------------")
	}
	fmt.Println("===================")

	results = indexer.Lookup(must, should, not, true, nil, onFlag, offFlag, orFlags)
	fmt.Println("有must无should，有序，搜索结果：") //talent1和talent2被召回，且talent2排在前面
	for _, doc := range results {
		fmt.Println(doc)
		t := DeserializeTalent(data.Entry)
		fmt.Println(t.Id)
		fmt.Println(t.Name)
		fmt.Println(t.Age)
		fmt.Println(t.Is211)
		fmt.Println("------------------")
	}
	fmt.Println("===================")

	not = []*types.Keyword{kw4}
	results = indexer.Lookup(must, should, not, false, nil, onFlag, offFlag, orFlags)
	fmt.Println("搜索结果：") //talent1被召回
	for _, doc := range results {
		fmt.Println(doc)
		t := DeserializeTalent(data.Entry)
		fmt.Println(t.Id)
		fmt.Println(t.Name)
		fmt.Println(t.Age)
		fmt.Println(t.Is211)
		fmt.Println("------------------")
	}
	fmt.Println("===================")

	not = []*types.Keyword{}
	should = []*types.Keyword{kw4, kw1}

	results = indexer.Lookup(nil, should, not, false, nil, onFlag, offFlag, orFlags)
	fmt.Println("无must有should，无序，搜索结果：") //talent2被召回
	for _, doc := range results {
		fmt.Println(doc)
		t := DeserializeTalent(data.Entry)
		fmt.Println(t.Id)
		fmt.Println(t.Name)
		fmt.Println(t.Age)
		fmt.Println(t.Is211)
		fmt.Println("------------------")
	}
	fmt.Println("===================")

	should = []*types.Keyword{}
	filter[uint32(talent2.Id)] = true
	results = indexer.Lookup(must, should, not, false, nil, onFlag, offFlag, orFlags)
	fmt.Println("搜索结果：") //talent1被召回
	for _, doc := range results {
		fmt.Println(doc)
		t := DeserializeTalent(data.Entry)
		fmt.Println(t.Id)
		fmt.Println(t.Name)
		fmt.Println(t.Age)
		fmt.Println(t.Is211)
		fmt.Println("------------------")
	}
	fmt.Println("===================")

	indexer.RemoveDoc(uint32(talent1.Id))
	indexer.RemoveDoc(uint32(talent2.Id))
	time.Sleep(100 * time.Millisecond) //稍等一会儿，等Delete操作执行完成
	indexer.Lookup(must, should, not, false, nil, onFlag, offFlag, orFlags)
	// fmt.Println("把待删除的彻底删除")
	// indexer.CleanToBeDeleteBuffer()
	results = indexer.Lookup(must, should, not, false, nil, onFlag, offFlag, orFlags)
	if len(results) == 0 {
		fmt.Println("没有搜索结果")
	} else {
		for _, doc := range results {
			fmt.Println(doc)
			t := DeserializeTalent(data.Entry)
			fmt.Println(t.Id)
			fmt.Println(t.Name)
			fmt.Println(t.Age)
			fmt.Println(t.Is211)
			fmt.Println("------------------")
		}
		t.Fail()
	}
	fmt.Println("===================")
}

func TestVint(t *testing.T) {
	t.SkipNow()

	var num int64 = 222222
	byteDatas := make([]byte, 8)
	cnt := binary.PutVarint(byteDatas, num)
	fmt.Println(byteDatas[:cnt])

	brr := []byte{142, 200, 13}
	if num, cnt := binary.Varint(brr); cnt > 0 {
		fmt.Println(num)
	}

	arr := []byte{142, 200, 13, 0}
	byteReader := bytes.NewReader(arr)
	for {
		if docId, err := binary.ReadVarint(byteReader); err == nil {
			k := strconv.FormatInt(docId, 10)
			fmt.Println(k)
		} else {
			break
		}
	}
}

func TestClose(t *testing.T) {

}

// go test -v .\inverted_index\test\
