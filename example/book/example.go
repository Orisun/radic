package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/Orisun/radic"
	"github.com/Orisun/radic/types"
)

//Book 以图书搜索为例
type Book struct {
	Id      uint32
	Name    string
	Price   float32
	Chinese bool //是否为汉语版
}

//Serialize 图书序列化。序列化和反序列化由调用方决定，这不是radic负责的范畴。
func (book *Book) Serialize() []byte {
	var value bytes.Buffer
	encoder := gob.NewEncoder(&value) //gob是go自带的序列化方法，当然也可以用protobuf等其它方式
	err := encoder.Encode(book)
	if err == nil {
		return value.Bytes()
	} else {
		fmt.Println(err)
		return []byte{}
	}
}

//DeserializeBook  图书反序列化
func DeserializeBook(v []byte) *Book {
	buf := bytes.NewReader(v)
	dec := gob.NewDecoder(buf)
	var data = Book{}
	err := dec.Decode(&data)
	if err == nil {
		return &data
	} else {
		fmt.Println(err)
		return nil
	}
}

func main() {
	/**
	初始化索引
	*/
	options := types.IndexerOpts{
		IndexName:             "book",
		AddCacheSize:          500,
		DeleteCacheSize:       100,
		InvertIndexSegmentNum: 16,
		ForwardIndexSize:      100000,
		DbShardNum:            8,
		DbPath:                "db",
		DbType:                "badger",
		AddDocThreads:         8,
		DeleteDocThreads:      4,
	}

	engine := radic.IndexEngine{}
	engine.Init(options)
	defer engine.Destroy()

	/**
	往索引上添加数据
	*/
	book1 := Book{Id: 1, Name: "工业机器学习算法详解与实战", Price: 100.0, Chinese: true}
	book2 := Book{Id: 2, Name: "effective go", Price: 200.0, Chinese: false}
	kw1 := &types.Keyword{Field: "field", Word: "算法"}
	kw2 := &types.Keyword{Field: "title", Word: "工业"}
	kw3 := &types.Keyword{Field: "title", Word: "机器学习"}
	kw4 := &types.Keyword{Field: "field", Word: "编程"}
	kw5 := &types.Keyword{Field: "title", Word: "go"}
	docInfo1 := types.DocInfo{
		DocId:            uint32(book1.Id),
		Keyword:          []*types.Keyword{kw1, kw2, kw3},
		RankScore:        book1.Price,
		Entry:            book1.Serialize(),
		CompositeFeature: 1, //转成二进制，倒数第1位上是1
	}
	docInfo2 := types.DocInfo{
		DocId:            uint32(book2.Id),
		Keyword:          []*types.Keyword{kw4, kw5},
		RankScore:        book2.Price,
		Entry:            book2.Serialize(),
		CompositeFeature: 3, //转成二进制，倒数第1位和第2位上都是1
	}
	engine.IndexDoc(docInfo1) //往索引上添加数据是异步执行的
	engine.IndexDoc(docInfo2)
	time.Sleep(100 * time.Millisecond) //稍等一会儿，等Add操作执行完成
	engine.InvertIndexStatistics("")   //建好索引后最好调用一次InvertIndexStatistics，有利于搜索速度的提升

	/**
	按关键词检索
	*/
	keyword := &types.Keyword{Field: "title", Word: "go"}
	request := types.SearchRequest{
		Must:          []*types.Keyword{keyword},
		Should:        nil,
		Not:           nil,
		OutputOffset:  0,
		OnFlag:        1, //要求倒数第1位上是1
		OffFlag:       4, //要求倒数第3位上不能是1
		Orderless:     false,
		CountDocsOnly: false,
		Timeout:       200,
	}
	response := engine.Search(request)
	if !response.Timeout {
		fmt.Printf("一共有%d条搜索结果:\n", response.Total)
		for _, doc := range response.Docs {
			book := DeserializeBook(doc)
			fmt.Printf("book name %s\n", book.Name)
		}
	}
	fmt.Println("=======================================")

	/**
	按doc id检索
	*/
	doc := engine.GetDoc(book1.Id, 1) //根据ID获取一个doc的详情。flag=1仅获取detail，flag=0都获取
	book := DeserializeBook(doc.Entry)
	fmt.Printf("book name %s\n", book.Name)
	fmt.Printf("book keywords %v\n", doc.Keyword) //由于flag=1，所以获取不到Keyword
	fmt.Println("=======================================")

	/**
	删除doc
	*/
	engine.RemoveDoc(book1.Id)

	/**
	遍历索引上的数据
	*/
	engine.IterIndex(func(docId uint32, docInfo types.DocInfo) error {
		book := DeserializeBook(docInfo.Entry)
		fmt.Printf("book name %s\n", book.Name)
		return nil
	})
	fmt.Println("=======================================")
}
