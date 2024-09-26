package video_search

import (
	"reflect"
	"sync"
	"time"

	demo "github.com/Orisun/radic/v2/example"
	"github.com/Orisun/radic/v2/example/video_search/common"
	"github.com/Orisun/radic/v2/example/video_search/filter"
	"github.com/Orisun/radic/v2/example/video_search/recaller"
	"github.com/Orisun/radic/v2/util"
	"golang.org/x/exp/maps"
)

type Recaller interface {
	Recall(*common.VideoSearchContext) []*demo.BiliVideo
}

type Filter interface {
	Apply(*common.VideoSearchContext)
}

// 模板方法Template Method模式。超类
type VideoSearcher struct {
	Recallers []Recaller //实际中，除了正常的关键词召回外，可能还要召回广告
	Filters   []Filter
}

// Builder模式
func (searcher *VideoSearcher) WithRecaller(recaller ...Recaller) {
	searcher.Recallers = append(searcher.Recallers, recaller...)
}

func (searcher *VideoSearcher) WithFilter(filter ...Filter) {
	searcher.Filters = append(searcher.Filters, filter...)
}

func (searcher *VideoSearcher) Recall(searchContext *common.VideoSearchContext) {
	if len(searcher.Recallers) == 0 {
		return
	}
	//并行执行多路召回
	collection := make(chan *demo.BiliVideo, 1000)
	wg := sync.WaitGroup{}
	wg.Add(len(searcher.Recallers))
	for _, recaller := range searcher.Recallers {
		go func(recaller Recaller) {
			defer wg.Done()
			rule := reflect.TypeOf(recaller).Name()
			result := recaller.Recall(searchContext)
			util.Log.Printf("recall %d docs by %s", len(result), rule)
			for _, video := range result {
				collection <- video
			}
		}(recaller)
	}
	//通过map合并多路召回的结果
	videoMap := make(map[string]*demo.BiliVideo, 1000)
	receiveFinish := make(chan struct{})
	go func() {
		for {
			video, ok := <-collection
			if !ok {
				break
			}
			videoMap[video.Id] = video
		}
		receiveFinish <- struct{}{}
	}()
	wg.Wait()
	close(collection)
	<-receiveFinish

	searchContext.Videos = maps.Values(videoMap)
}

func (searcher *VideoSearcher) Filter(searchContext *common.VideoSearchContext) {
	//顺序执行各个过滤规则
	for _, filter := range searcher.Filters {
		filter.Apply(searchContext)
	}
}

// 超类定义了一个算法的框架，在子类中重写特定的算法步骤（即recall和filter这2步）
func (searcher *VideoSearcher) Search(searchContext *common.VideoSearchContext) []*demo.BiliVideo {
	t1 := time.Now()
	//召回
	searcher.Recall(searchContext)
	t2 := time.Now()
	util.Log.Printf("recall %d docs in %d ms", len(searchContext.Videos), t2.Sub(t1).Milliseconds())
	//过滤
	searcher.Filter(searchContext)
	t3 := time.Now()
	util.Log.Printf("after filter remain %d docs in %d ms", len(searchContext.Videos), t3.Sub(t2).Milliseconds())
	return searchContext.Videos
}

// 子类
type AllVideoSearcher struct {
	VideoSearcher
}

func NewAllVideoSearcher() *AllVideoSearcher {
	searcher := new(AllVideoSearcher)
	searcher.WithRecaller(recaller.KeywordRecaller{})
	searcher.WithFilter(filter.ViewFilter{})
	return searcher
}

// 子类
type UpVideoSearcher struct {
	VideoSearcher
}

func NewUpVideoSearcher() *UpVideoSearcher {
	searcher := new(UpVideoSearcher)
	searcher.WithRecaller(recaller.KeywordAuthorRecaller{})
	searcher.WithFilter(filter.ViewFilter{})
	return searcher
}
