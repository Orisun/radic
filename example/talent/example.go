// Date         : 2018/7/30 下午8:10
// Author       : zhangchaoyang
// Description  :
package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/Orisun/radic"
	"github.com/Orisun/radic/types"
)

type Talent struct {
	Id    uint32
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

type ManualRankCriteria struct {
}

func (criteria ManualRankCriteria) Score(s float32) float32 {
	return s
}

func main() {
	options := types.IndexerOpts{
		IndexName:             "talent",
		AddCacheSize:          500,
		DeleteCacheSize:       100,
		InvertIndexSegmentNum: 16,
		ForwardIndexSize:      100000,
		DbShardNum:            8,
		DbPath:                "db",
		DbType:                "badger",
		AddDocThreads:         8,
		DeleteDocThreads:      4,
		MaxRetrieve:           2, //搜索时最多返回2条结果。RankScore最小有会被丢弃
	}
	engine := radic.IndexEngine{}
	engine.Start(options)
	defer engine.Stop()

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
	talent3 := Talent{
		Id:    33333333,
		Age:   30,
		Name:  "朝阳",
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
		RankScore: 15,
		Entry:     talent1.Serialize(),
	}
	docInfo2 := types.DocInfo{
		DocId:     uint32(talent2.Id),
		Keyword:   []*types.Keyword{kw1, kw4, kw5, kw6},
		RankScore: 8, //它是三者当中RankScore最小的
		Entry:     talent2.Serialize(),
	}
	docInfo3 := types.DocInfo{
		DocId:     uint32(talent3.Id),
		Keyword:   []*types.Keyword{kw1, kw4, kw5, kw6},
		RankScore: 9,
		Entry:     talent3.Serialize(),
	}
	engine.IndexDoc(docInfo1) //往索引上添加数据是异步执行的
	engine.IndexDoc(docInfo2)
	engine.IndexDoc(docInfo3)
	time.Sleep(100 * time.Millisecond)      //稍等一会儿，等Add操作执行完成
	engine.InvertIndexStatistics("sta.txt") //建好索引后最好调用一次InvertIndexStatistics，有利于搜索速度的提升
	engine.Stop()
	engine.Start(options)
	engine.LoadDataFromDb2InvertIndex()

	doc := engine.GetDoc(talent1.Id, 1) //仅获取detail
	// fmt.Println(doc)
	t2 := DeserializeTalent(doc.Entry)
	// fmt.Println(t2)
	fmt.Printf("name of talent %d is %s, keywords %v\n", talent1.Id, t2.Name, doc.Keyword)
	fmt.Println("================================")
	doc = engine.GetDoc(talent1.Id, 0) //获取detail和keywords
	t2 = DeserializeTalent(doc.Entry)
	fmt.Printf("name of talent %d is %s, keywords %v\n", talent1.Id, t2.Name, doc.Keyword)
	fmt.Println("================================")

	docs := engine.GetDocs([]uint32{talent1.Id, talent2.Id})
	for _, doc := range docs {
		talent := DeserializeTalent(doc.Entry)
		fmt.Printf("name of talent %d is %s\n", talent.Id, talent.Name)
	}
	fmt.Println("================================")

	engine.IterIndex(func(docId uint32, docInfo types.DocInfo) error {
		talent := DeserializeTalent(docInfo.Entry)
		fmt.Printf("name of talent %d is %s\n", talent.Id, talent.Name)
		return nil
	})
	fmt.Println("================================")

	keyword := &types.Keyword{Field: "company", Word: "百度"}
	request := types.SearchRequest{
		Must:          []*types.Keyword{keyword},
		Should:        nil,
		Not:           nil,
		OutputOffset:  0,
		Orderless:     false,
		CountDocsOnly: false,
		Timeout:       200,
	}
	response := engine.Search(request)
	if !response.Timeout {
		fmt.Printf("一共有%d条搜索结果\n", response.Total)
		for _, doc := range response.Docs {
			// fmt.Println(doc)
			//fmt.Println(doc.HitKeywords)
			talent := DeserializeTalent(doc)
			fmt.Printf("name of talent %d is %s\n", talent.Id, talent.Name)
			fmt.Println("-------------------------")
		}
	}
	fmt.Println("================================")

	engine.InvertIndexStatistics("sta.txt")

	engine.RemoveDoc(talent1.Id) //从索引上删除数据是异步执行的
	engine.RemoveDoc(talent2.Id)
	time.Sleep(100 * time.Millisecond) //稍等一会儿，等Remove操作执行完成

	engine.IterIndex(func(docId uint32, docInfo types.DocInfo) error {
		talent := DeserializeTalent(docInfo.Entry)
		fmt.Printf("name of talent %d is %s\n", talent.Id, talent.Name)
		return nil
	})
	fmt.Println("================================")

	response = engine.Search(request)
	if !response.Timeout {
		fmt.Printf("一共有%d条搜索结果\n", response.Total)
		for _, doc := range response.Docs {
			talent := DeserializeTalent(doc)
			fmt.Printf("name of talent %d is %s\n", talent.Id, talent.Name)
		}
	}
	fmt.Println("================================")
}

// go run .\example\talent
