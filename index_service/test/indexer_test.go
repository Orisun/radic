package test

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strings"
	"testing"

	"github.com/Orisun/radic/v2/index_service"
	"github.com/Orisun/radic/v2/types"
	"github.com/Orisun/radic/v2/util"
)

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
		fmt.Println("序列化图书失败", err)
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
		fmt.Println("反序列化图书失败", err)
		return nil
	}
}

var (
	// dbType=kvdb.BOLT
	// dbPath=util.RootPath+"data/local_db/book_bolt"
	dbType = types.BADGER
	dbPath = util.RootPath + "data/local_db/book_badger"
)

func TestSearch(t *testing.T) {
	radic := new(index_service.Indexer)
	if err := radic.Init(100, dbType, dbPath); err != nil {
		fmt.Println(err)
		t.Fail()
		return
	}
	defer radic.Close()

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

	doc1 := types.Document{
		Id:          book1.ISBN,
		BitsFeature: 0b10101, //二进制
		Keywords:    []*types.Keyword{{Field: "content", Word: "机器学习"}, {Field: "content", Word: "神经网络"}, {Field: "title", Word: book1.Title}},
		Bytes:       book1.Serialize(), //写入索引时需要自行序列化
	}
	doc2 := types.Document{
		Id:          book2.ISBN,
		BitsFeature: 0b10011, //二进制
		Keywords:    []*types.Keyword{{Field: "content", Word: "唐朝"}, {Field: "content", Word: "文物"}, {Field: "title", Word: book2.Title}},
		Bytes:       book2.Serialize(),
	}
	doc3 := types.Document{
		Id:          book3.ISBN,
		BitsFeature: 0b11101, //二进制
		Keywords:    []*types.Keyword{{Field: "content", Word: "动物"}, {Field: "content", Word: "文物"}, {Field: "title", Word: book3.Title}},
		Bytes:       book3.Serialize(),
	}

	radic.AddDoc(doc1)
	radic.AddDoc(doc2)
	radic.AddDoc(doc3)

	q1 := types.NewTermQuery("title", "生命起源")
	q2 := types.NewTermQuery("content", "文物")
	q3 := types.NewTermQuery("title", "中国历史")
	q4 := types.NewTermQuery("content", "文物")
	q5 := types.NewTermQuery("content", "唐朝")

	q6 := q1.And(q2)
	q7 := q3.And(q4).And(q5)

	q8 := q6.Or(q7)

	var onFlag uint64 = 0b10000
	var offFlag uint64 = 0b01000
	orFlags := []uint64{uint64(0b00010), uint64(0b00101)}
	docs := radic.Search(q8, onFlag, offFlag, orFlags) //检索
	for _, doc := range docs {
		book := DeserializeBook(doc.Bytes) //检索的结果是二进流，需要自反序列化
		if book != nil {
			fmt.Printf("%s %s %s %.1f\n", book.ISBN, book.Title, book.Author, book.Price)
		}
	}
	fmt.Println(strings.Repeat("-", 50))

	radic.DeleteDoc(doc2.Id)
	docs = radic.Search(q8, onFlag, offFlag, orFlags) //检索
	for _, doc := range docs {
		book := DeserializeBook(doc.Bytes) //检索的结果是二进流，需要自反序列化
		if book != nil {
			fmt.Printf("%s %s %s %.1f\n", book.ISBN, book.Title, book.Author, book.Price)
		}
	}
	fmt.Println(strings.Repeat("-", 50))

	radic.AddDoc(doc2)
	docs = radic.Search(q8, onFlag, offFlag, orFlags) //检索
	for _, doc := range docs {
		book := DeserializeBook(doc.Bytes) //检索的结果是二进流，需要自反序列化
		if book != nil {
			fmt.Printf("%s %s %s %.1f\n", book.ISBN, book.Title, book.Author, book.Price)
		}
	}
	fmt.Println(strings.Repeat("-", 50))

}

func TestLoadFromIndexFile(t *testing.T) {
	indexer := new(index_service.Indexer)
	if err := indexer.Init(100, dbType, dbPath); err != nil {
		fmt.Println(err)
		t.Fail()
		return
	}
	defer indexer.Close()

	n := indexer.LoadFromIndexFile()
	if n == 0 {
		return
	} else {
		util.Log.Printf("load %d document from invere document", n)
	}

	q1 := types.NewTermQuery("title", "生命起源")
	q2 := types.NewTermQuery("content", "文物")
	q3 := types.NewTermQuery("title", "中国历史")
	q4 := types.NewTermQuery("content", "文物")
	q5 := types.NewTermQuery("content", "唐朝")

	q6 := q1.And(q2)
	q7 := q3.And(q4).And(q5)

	q8 := q6.Or(q7)

	var onFlag uint64 = 0b10000
	var offFlag uint64 = 0b01000
	orFlags := []uint64{uint64(0b00010), uint64(0b00101)}
	docs := indexer.Search(q8, onFlag, offFlag, orFlags) //检索
	for _, doc := range docs {
		book := DeserializeBook(doc.Bytes) //检索的结果是二进流，需要自反序列化
		if book != nil {
			fmt.Printf("%s %s %s %.1f\n", book.ISBN, book.Title, book.Author, book.Price)
		}
	}
	fmt.Println(strings.Repeat("-", 50))

	book2 := Book{
		ISBN:    "436246383",
		Title:   "中国历史",
		Author:  "李四",
		Price:   39.0,
		Content: "冰雪奇缘2 中文版电影原声带 (Frozen 2 (Mandarin Original Motion Picture",
	}
	doc2 := types.Document{
		Id:          book2.ISBN,
		BitsFeature: 0b10011, //二进制
		Keywords:    []*types.Keyword{{Field: "content", Word: "唐朝"}, {Field: "content", Word: "文物"}, {Field: "title", Word: book2.Title}},
		Bytes:       book2.Serialize(),
	}

	indexer.DeleteDoc(doc2.Id)
	docs = indexer.Search(q8, onFlag, offFlag, orFlags) //检索
	for _, doc := range docs {
		book := DeserializeBook(doc.Bytes) //检索的结果是二进流，需要自反序列化
		if book != nil {
			fmt.Printf("%s %s %s %.1f\n", book.ISBN, book.Title, book.Author, book.Price)
		}
	}
	fmt.Println(strings.Repeat("-", 50))

	indexer.AddDoc(doc2)
	docs = indexer.Search(q8, onFlag, offFlag, orFlags) //检索
	for _, doc := range docs {
		book := DeserializeBook(doc.Bytes) //检索的结果是二进流，需要自反序列化
		if book != nil {
			fmt.Printf("%s %s %s %.1f\n", book.ISBN, book.Title, book.Author, book.Price)
		}
	}
	fmt.Println(strings.Repeat("-", 50))
}

// go test -v ./index_service/test -run=^TestSearch$ -count=1
// go test -v ./index_service/test -run=^TestLoadFromIndexFile$ -count=1
