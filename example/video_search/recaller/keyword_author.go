package recaller

import (
	"strings"

	demo "github.com/Orisun/radic/v2/example"
	"github.com/Orisun/radic/v2/example/video_search/common"
	"github.com/Orisun/radic/v2/types"
	"github.com/gogo/protobuf/proto"
)

type KeywordAuthorRecaller struct {
}

func (KeywordAuthorRecaller) Recall(ctx *common.VideoSearchContext) []*demo.BiliVideo {
	request := ctx.Request
	if request == nil {
		return nil
	}
	indexer := ctx.Indexer
	if indexer == nil {
		return nil
	}
	keywords := request.Keywords
	query := new(types.TermQuery)
	if len(keywords) > 0 {
		for _, word := range keywords {
			query = query.And(types.NewTermQuery("content", word)) //满足关键词
		}
	}
	v := ctx.Ctx.Value(common.UN("user_name"))
	if v != nil {
		if author, ok := v.(string); ok {
			if len(author) > 0 {
				query = query.And(types.NewTermQuery("author", strings.ToLower(author))) //满足作者
			}
		}
	}

	orFlags := []uint64{demo.GetClassBits(request.Classes)} //满足类别
	docs := indexer.Search(query, 0, 0, orFlags)
	videos := make([]*demo.BiliVideo, 0, len(docs))
	for _, doc := range docs {
		var video demo.BiliVideo
		if err := proto.Unmarshal(doc.Bytes, &video); err == nil {
			videos = append(videos, &video)
		}
	}
	return videos
}
