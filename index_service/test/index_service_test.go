package test

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/Orisun/radic/v2/index_service"
	"github.com/Orisun/radic/v2/types"
	"github.com/Orisun/radic/v2/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	servicePort = 5678
)

// 启动grpc server
func StartService() {
	// 监听本地的5678端口
	lis, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(servicePort))
	if err != nil {
		panic(err)
	}

	server := grpc.NewServer()
	service := new(index_service.IndexServiceWorker)
	service.Init(50000, types.BADGER, util.RootPath+"data/local_db/book_badger") //不进行服务注册，client直连server
	service.Indexer.LoadFromIndexFile()                                          //从文件中加载索引数据
	// 注册服务的具体实现
	index_service.RegisterIndexServiceServer(server, service)
	go func() {
		// 启动服务
		fmt.Printf("start grpc server on port %d\n", servicePort)
		err = server.Serve(lis) //Serve会一直阻塞，所以放到一个协程里异步执行
		if err != nil {
			panic(err)
		}
	}()
}

func TestIndexService(t *testing.T) {
	StartService()              //server和client分到不同的协程里去。实际中，server和client是部署在不同的机器上
	time.Sleep(1 * time.Second) //等server启动完毕

	//连接到服务端
	conn, err := grpc.DialContext(
		context.Background(),
		"127.0.0.1:"+strconv.Itoa(servicePort),
		grpc.WithTransportCredentials(insecure.NewCredentials()), //Credential即使为空，也必须设置
	)
	if err != nil {
		fmt.Printf("dial failed: %s", err)
		return
	}
	//创建client
	client := index_service.NewIndexServiceClient(conn)

	//测试Search接口
	query := types.NewTermQuery("content", "文物")
	query = query.And(types.NewTermQuery("content", "唐朝"))
	request := &index_service.SearchRequest{
		Query: query,
	}
	result, err := client.Search(context.Background(), request)
	if err != nil {
		fmt.Println(err)
		t.Fail()
	} else {
		docId := ""
		for _, doc := range result.Results {
			book := DeserializeBook(doc.Bytes) //检索的结果是二进流，需要自反序列化
			if book != nil {
				fmt.Printf("%s %s %s %s %.1f\n", doc.Id, book.ISBN, book.Title, book.Author, book.Price)
				docId = doc.Id
			}
		}
		//测试Delete接口
		if len(docId) > 0 {
			affect, err := client.DeleteDoc(context.Background(), &index_service.DocId{DocId: docId})
			if err != nil {
				fmt.Println(err)
				t.Fail()
			} else {
				fmt.Printf("删除%d个doc\n", affect.Count)
			}
		}
		//测试Add接口
		book := Book{
			ISBN:    "436246383",
			Title:   "上下五千年",
			Author:  "李四",
			Price:   39.0,
			Content: "冰雪奇缘2 中文版电影原声带 (Frozen 2 (Mandarin Original Motion Picture",
		}
		doc := types.Document{
			Id:          book.ISBN,
			BitsFeature: 0b10011, //二进制
			Keywords:    []*types.Keyword{{Field: "content", Word: "唐朝"}, {Field: "content", Word: "文物"}, {Field: "title", Word: book.Title}},
			Bytes:       book.Serialize(),
		}
		affect, err := client.AddDoc(context.Background(), &doc)
		if err != nil {
			fmt.Println(err)
			t.Fail()
		} else {
			fmt.Printf("添加%d个doc\n", affect.Count)
		}
		//测试Search接口
		request := &index_service.SearchRequest{
			Query: query,
		}
		result, err := client.Search(context.Background(), request)
		if err != nil {
			fmt.Println(err)
			t.Fail()
		} else {
			for _, doc := range result.Results {
				book := DeserializeBook(doc.Bytes) //检索的结果是二进流，需要自反序列化
				if book != nil {
					fmt.Printf("%s %s %s %s %.1f\n", doc.Id, book.ISBN, book.Title, book.Author, book.Price)
				}
			}
		}
	}
}

// go test -v ./index_service/test -run=^TestIndexService$ -count=1
