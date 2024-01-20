package handler

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Orisun/radic/v2/demo"
	"github.com/Orisun/radic/v2/demo/video_search"
	"github.com/Orisun/radic/v2/demo/video_search/filter"
	"github.com/Orisun/radic/v2/demo/video_search/recaller"
	"github.com/Orisun/radic/v2/index_service"
	"github.com/Orisun/radic/v2/internal/kvdb"
	"github.com/Orisun/radic/v2/types"
	"github.com/Orisun/radic/v2/util"
	"github.com/gogo/protobuf/proto"

	"github.com/gin-gonic/gin"
)

var (
	dbType  = kvdb.BOLT
	dbPath  = util.RootPath + "data/local_db/vedio_bolt"
	indexer *index_service.Indexer
	csvFile = util.RootPath + "data/bili_video.csv"
)

// 从CSV文件重建索引
func BuildIndex() {
	os.Remove(dbPath) //先删除原有的索引文件
	indexer = new(index_service.Indexer)
	if err := indexer.Init(50000, dbType, dbPath); err != nil {
		panic(err)
	}
	demo.BuildIndexFromFile(csvFile, indexer)
}

// 直接加载索引文件
func LoadIndex() {
	indexer = new(index_service.Indexer)
	if err := indexer.Init(50000, dbType, dbPath); err != nil {
		panic(err)
	}
	indexer.LoadFromIndexFile()
}

func CloseIndex() {
	indexer.Close()
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
	docs := indexer.Search(query, 0, 0, orFlags)
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

	searchCtx := &video_search.VideoSearchContext{
		Ctx:     context.Background(),
		Request: &request,
		Indexer: indexer,
	}
	searcher := video_search.VideoSearcher{}
	searcher.WithRecaller(recaller.KeywordRecaller{})
	searcher.WithFilter(filter.ViewFilter{})
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

	userName, err := url.QueryUnescape(ctx.Request.Header.Get("UserName"))
	if err != nil || len(userName) == 0 {
		ctx.String(http.StatusBadRequest, "获取不到登录用户名")
		return
	}
	searchCtx := &video_search.VideoSearchContext{
		Ctx:     context.WithValue(context.Background(), "user_name", userName), //从request header里获得user_name，再放到context里
		Request: &request,
		Indexer: indexer,
	}
	searcher := video_search.VideoSearcher{}
	searcher.WithRecaller(recaller.KeywordAuthorRecaller{})
	searcher.WithFilter(filter.ViewFilter{})
	videos := searcher.Search(searchCtx)

	ctx.JSON(http.StatusOK, videos) //把搜索结果以json形式返回给前端
}
