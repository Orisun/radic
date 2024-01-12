package demo

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Orisun/radic/v2"
	"github.com/Orisun/radic/v2/internal/kvdb"
	"github.com/Orisun/radic/v2/types"
	"github.com/Orisun/radic/v2/util"
	"github.com/gogo/protobuf/proto"

	"github.com/gin-gonic/gin"
)

type SearchRequest struct {
	Author   string
	Classes  []string //类别，命中一个即可
	Keywords []string //关键词，必须全部命中
	ViewFrom int      //视频播放量下限
	ViewTo   int      //视频播放量上限
}

var (
	dbType  = kvdb.BOLT
	dbPath  = util.RootPath + "data/local_db/vedio_bolt"
	indexer *radic.Indexer
	csvFile = util.RootPath + "data/bili_video.csv"
)

// 从CSV文件重建索引
func BuildIndex() {
	os.Remove(dbPath) //先删除原有的索引文件
	indexer = new(radic.Indexer)
	if err := indexer.Init(50000, dbType, dbPath); err != nil {
		panic(err)
	}
	BuildIndexFromFile(csvFile, indexer)
}

// 直接加载索引文件
func LoadIndex() {
	indexer = new(radic.Indexer)
	if err := indexer.Init(50000, dbType, dbPath); err != nil {
		panic(err)
	}
	indexer.LoadFromIndexFile()
}

func CloseIndex() {
	indexer.Close()
}

// 搜索接口
func Search(ctx *gin.Context) {
	var request SearchRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		log.Printf("bind request parameter failed: %s", err)
		ctx.String(http.StatusBadRequest, "invalid json")
		return
	}

	keywords := make([]string, 0, len(request.Keywords))
	for _, w := range request.Keywords {
		word := strings.TrimSpace(strings.ToLower(w))
		if len(word) > 0 {
			keywords = append(keywords, word)
		}
	}
	fmt.Println(keywords)

	if len(keywords) == 0 && len(request.Author) == 0 {
		ctx.String(http.StatusBadRequest, "关键词和作者不能同时为空")
		return
	}
	query := types.TermQuery{}
	if len(keywords) > 0 {
		for _, word := range keywords {
			query = query.And(types.NewTermQuery("content", word)) //满足关键词
		}
	}
	if len(request.Author) > 0 {
		query = query.And(types.NewTermQuery("author", strings.ToLower(request.Author))) //满足作者
	}
	orFlags := []uint64{GetClassBits(request.Classes)} //满足类别
	docs := indexer.Search(query, 0, 0, orFlags)
	videos := make([]BiliVideo, 0, len(docs))
	for _, doc := range docs {
		var video BiliVideo
		if err := proto.Unmarshal(doc, &video); err == nil {
			if video.View >= int32(request.ViewFrom) && (request.ViewTo <= 0 || video.View <= int32(request.ViewTo)) { //满足播放量的区间范围
				videos = append(videos, video)
			}
		}
	}
	ctx.JSON(http.StatusOK, videos) //把搜索结果以json形式返回给前端
}
