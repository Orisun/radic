package radic

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"testing"

	"github.com/Orisun/radic/v2/internal/kvdb"
	"github.com/Orisun/radic/v2/types"
	"github.com/Orisun/radic/v2/util"
)

func TestInt2Varint(t *testing.T) {
	fmt.Println(Int2Varint(100))
	fmt.Println(Int2Varint(1000))
	fmt.Println(Int2Varint(10000))
	var i uint32
	for i = 1; i < 100000000; i++ {
		for _, ele := range Int2Varint(i) {
			if ele == 0 {
				fmt.Println(i)
				t.Fail()
			}
		}
	}
}

type Book struct {
	ISBN    string
	Title   string
	Author  string
	Price   float64
	Content string
}

// Serialize 图书序列化。序列化和反序列化由调用方决定，这不是radic负责的范畴。
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

// DeserializeBook  图书反序列化
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

func TestSearch(t *testing.T) {
	indexer := new(Indexer)
	if err := indexer.Init(100, kvdb.BOLT, util.RootPath+"data/local_db/book_bolt"); err != nil {
		fmt.Println(err)
		t.Fail()
		return
	}
	defer indexer.Close()

	book1 := Book{
		ISBN:    "315246546",
		Title:   "计算机系列丛书",
		Author:  "张三",
		Price:   59.0,
		Content: "冰雪奇缘2 中文版电影原声带 (Frozen 2 (Mandarin Original Motion Picture",
	}
	book2 := Book{
		ISBN:    "436246383",
		Title:   "中国历史",
		Author:  "李四",
		Price:   39.0,
		Content: "冰雪奇缘2 中文版电影原声带 (Frozen 2 (Mandarin Original Motion Picture",
	}
	book3 := Book{
		ISBN:    "54325435634",
		Title:   "生命起源",
		Author:  "赵六",
		Price:   49.0,
		Content: "冰雪奇缘2 中文版电影原声带 (Frozen 2 (Mandarin Original Motion Picture",
	}

	doc1 := types.DocInfo{
		DocId:            1,
		CompositeFeature: 0b10101, //二进制
		Keyword:          []*types.Keyword{{Field: "content", Word: "机器学习"}, {Field: "content", Word: "神经网络"}, {Field: "title", Word: "计算机"}},
		Entry:            book1.Serialize(), //写入索引时需要自行序列化
	}
	doc2 := types.DocInfo{
		DocId:            2,
		CompositeFeature: 0b10011, //二进制
		Keyword:          []*types.Keyword{{Field: "content", Word: "唐朝"}, {Field: "content", Word: "文物"}, {Field: "title", Word: "历史"}},
		Entry:            book2.Serialize(),
	}
	doc3 := types.DocInfo{
		DocId:            3,
		CompositeFeature: 0b11101, //二进制
		Keyword:          []*types.Keyword{{Field: "content", Word: "动物"}, {Field: "content", Word: "细胞"}, {Field: "title", Word: "生命"}},
		Entry:            book3.Serialize(),
	}

	indexer.AddDoc(doc1)
	indexer.AddDoc(doc2)
	indexer.AddDoc(doc3)

	q1 := types.NewTermQuery("title", "生命")
	q2 := types.NewTermQuery("content", "细胞")
	q3 := types.NewTermQuery("title", "历史")
	q4 := types.NewTermQuery("content", "文物")
	q5 := types.NewTermQuery("content", "唐朝")

	q6 := q1.And(q2)
	q7 := q3.And(q4).And(q5)

	q8 := q6.Or(q7)

	var onFlag uint32 = 0b10000
	var offFlag uint32 = 0b01000
	orFlags := []uint32{uint32(0b00010), uint32(0b00101)}
	docs := indexer.Search(q8, onFlag, offFlag, orFlags) //检索
	for _, doc := range docs {
		book := DeserializeBook(doc) //检索的结果是二进流，需要自反序列化
		if book != nil {
			fmt.Printf("%s %s %s %.1f\n", book.ISBN, book.Title, book.Author, book.Price)
		}
	}
}

// go test -v . -run=^TestInt2Varint$ -count=1
// go test -v . -run=^TestSearch$ -count=1
