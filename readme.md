go语言实现搜索引擎的索引部分，包括：<br>
[倒排索引](inverted_index/readme.md)<br>
[正排索引](forward_index/readme.md)<br>
[分布式索引](distributed/readme.md)<br>
<br>
我专门在腾讯课堂上开了门长达16小时的视频课堂：[《用Go实现工业级搜索引擎》](https://ke.qq.com/course/3617680)<br>

### 使用举例
```gotemplate
package main

import (
	"radic/types"
	"bytes"
	"encoding/gob"
	"fmt"
	"radic"
	"time"
)

//Book 以图书搜索为例
type Book struct {
	Id      uint32
	Name    string
	Price   float32
	Chinese bool //是否为汉语版
}

//Serialize 图书序列化。序列化和反序列化由调用方决定，这不是radic负责的范畴。
func (self *Book) Serialize() []byte {
	var value bytes.Buffer
	encoder := gob.NewEncoder(&value) //gob是go自带的序列化方法，当然也可以用protobuf等其它方式
	err := encoder.Encode(self)
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
	options := types.IndexerOpts{}
	engine := radic.IndexEngine{}
	engine.Init(options, types.DistOpts{}, "radic.log") //DistOpts为空，则采用单机索引
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
	fmt.Println()
	
	/**
	按doc id检索
	 */
	doc := engine.GetDoc(book1.Id, 1) //根据ID获取一个doc的详情。flag=1仅获取detail，flag=0都获取
	book := DeserializeBook(doc.Entry)
	fmt.Printf("book name %s\n", book.Name)
	fmt.Printf("book keywords %v\n", doc.Keyword) //由于flag=1，所以获取不到Keyword
	fmt.Println()
	
	/**
	删除doc
	 */
	engine.RemoveDoc(book2.Id)
	fmt.Println()
	
	/**
	遍历索引上的数据
	 */
	engine.IterIndex(func(docId uint32, docInfo types.DocInfo) error {
		book := DeserializeBook(docInfo.Entry)
		fmt.Printf("book name %s\n", book.Name)
		return nil
	})
}
```

更多API参见[radic.go](radic.go)
