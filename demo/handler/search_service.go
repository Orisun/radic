package handler

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Orisun/radic/v2/demo"
	"github.com/Orisun/radic/v2/demo/video_search"
	"github.com/Orisun/radic/v2/demo/video_search/common"
	"github.com/Orisun/radic/v2/index_service"
	"github.com/Orisun/radic/v2/internal/kvdb"
	"github.com/Orisun/radic/v2/types"
	"github.com/Orisun/radic/v2/util"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"

	"github.com/gin-gonic/gin"
)

var (
	dbType      = kvdb.BOLT
	dbPath      = util.RootPath + "data/local_db/vedio_bolt"
	csvFile     = util.RootPath + "data/bili_video.csv"
	etcdServers = []string{"127.0.0.1:2379"}
	service     *index_service.IndexServiceWorker
	indexer     index_service.IIndexer
)

// 从CSV文件重建索引
func BuildIndex() {
	os.Remove(dbPath) //先删除原有的索引文件
	indexer := new(index_service.Indexer)
	if err := indexer.Init(50000, dbType, dbPath); err != nil {
		panic(err)
	}
	demo.BuildIndexFromFile(csvFile, indexer)
	indexer.Close()
}

func StopWorker() {
	service.Close()
}

func StartWorker(port int) {
	// 监听本地端口
	lis, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		panic(err)
	}

	server := grpc.NewServer()
	service = new(index_service.IndexServiceWorker)
	service.Init(50000, dbType, dbPath+"_"+strconv.Itoa(port), etcdServers, port) //把自己注册到etcd
	service.LoadFromIndexFile()                                                   //从文件中加载索引数据
	// 注册服务的具体实现
	index_service.RegisterIndexServiceServer(server, service)
	go func(port int) {
		// 启动服务
		fmt.Printf("start grpc server on port %d\n", port)
		err = server.Serve(lis) //Serve会一直阻塞，所以放到一个协程里异步执行
		if err != nil {
			service.Close()
			fmt.Printf("start grpc server on port %d failed: %s\n", port, err)
		}
	}(port)
	// indexer=service.Indexer  //单机索引
	indexer = index_service.NewSentinel(etcdServers) //分布式索引
}

func clearnKeywords(words []string) []string {
	keywords := make([]string, 0, len(words))
	for _, w := range words {
		word := strings.TrimSpace(strings.ToLower(w))
		if len(word) > 0 {
			keywords = append(keywords, word)
		}
	}
	return keywords
}

// 搜索接口
func Search(ctx *gin.Context) {
	var request demo.SearchRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		log.Printf("bind request parameter failed: %s", err)
		ctx.String(http.StatusBadRequest, "invalid json")
		return
	}

	keywords := clearnKeywords(request.Keywords)
	if len(keywords) == 0 && len(request.Author) == 0 {
		ctx.String(http.StatusBadRequest, "关键词和作者不能同时为空")
		return
	}
	query := new(types.TermQuery)
	if len(keywords) > 0 {
		for _, word := range keywords {
			query = query.And(types.NewTermQuery("content", word)) //满足关键词
		}
	}
	if len(request.Author) > 0 {
		query = query.And(types.NewTermQuery("author", strings.ToLower(request.Author))) //满足作者
	}
	orFlags := []uint64{demo.GetClassBits(request.Classes)} //满足类别
	docs := service.Indexer.Search(query, 0, 0, orFlags)
	videos := make([]demo.BiliVideo, 0, len(docs))
	for _, doc := range docs {
		var video demo.BiliVideo
		if err := proto.Unmarshal(doc.Bytes, &video); err == nil {
			if video.View >= int32(request.ViewFrom) && (request.ViewTo <= 0 || video.View <= int32(request.ViewTo)) { //满足播放量的区间范围
				videos = append(videos, video)
			}
		}
	}
	ctx.JSON(http.StatusOK, videos) //把搜索结果以json形式返回给前端
}

// 搜索全站视频
func SearchAll(ctx *gin.Context) {
	var request demo.SearchRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		log.Printf("bind request parameter failed: %s", err)
		ctx.String(http.StatusBadRequest, "invalid json")
		return
	}

	request.Keywords = clearnKeywords(request.Keywords)
	if len(request.Keywords) == 0 && len(request.Author) == 0 {
		ctx.String(http.StatusBadRequest, "关键词和作者不能同时为空")
		return
	}

	searchCtx := &common.VideoSearchContext{
		Ctx:     context.Background(),
		Request: &request,
		Indexer: indexer,
	}
	searcher := video_search.NewAllVedioSearcher()
	videos := searcher.Search(searchCtx)

	ctx.JSON(http.StatusOK, videos) //把搜索结果以json形式返回给前端
}

// up主在后台搜索自己的视频
func SearchByAuthor(ctx *gin.Context) {
	var request demo.SearchRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		log.Printf("bind request parameter failed: %s", err)
		ctx.String(http.StatusBadRequest, "invalid json")
		return
	}

	request.Keywords = clearnKeywords(request.Keywords)
	if len(request.Keywords) == 0 {
		ctx.String(http.StatusBadRequest, "关键词不能为空")
		return
	}

	userName, ok := ctx.Value("user_name").(string) //从gin.Context里取得userName
	if !ok || len(userName) == 0 {
		ctx.String(http.StatusBadRequest, "获取不到登录用户名")
		return
	}
	searchCtx := &common.VideoSearchContext{
		Ctx:     context.WithValue(context.Background(), "user_name", userName), //把userName放到context里
		Request: &request,
		Indexer: indexer,
	}
	searcher := video_search.NewUpVedioSearcher()
	videos := searcher.Search(searchCtx)

	ctx.JSON(http.StatusOK, videos) //把搜索结果以json形式返回给前端
}
