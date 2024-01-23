package video_search

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/Orisun/radic/v2/demo"
	"github.com/Orisun/radic/v2/index_service"
	"github.com/Orisun/radic/v2/util"
	"golang.org/x/exp/maps"
)

type VideoSearchContext struct {
	Ctx     context.Context        //上下文参数
	Indexer index_service.IIndexer //索引。装饰器模式，搜索请求委派给的目标对象可能是本地的Indexer，也可能是分布式的Sentinel
	Request *demo.SearchRequest    //搜索请求
	Videos  []*demo.BiliVideo      //搜索结果
}

type Recaller interface {
	Recall(*VideoSearchContext) []*demo.BiliVideo
}

type Filter interface {
	Apply(*VideoSearchContext)
}

type VideoSearcher struct {
	Recallers []Recaller //实际中，除了正常的关键词召回外，可能还要召回广告
	Filters   []Filter
}

func (searcher *VideoSearcher) WithRecaller(recaller ...Recaller) {
	searcher.Recallers = append(searcher.Recallers, recaller...)
}

func (searcher *VideoSearcher) WithFilter(filter ...Filter) {
	searcher.Filters = append(searcher.Filters, filter...)
}

func (searcher *VideoSearcher) Recall(searchContext *VideoSearchContext) {
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
			util.Log.Printf("recall %d talents by %s", len(result), rule)
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

func (searcher *VideoSearcher) Filter(searchContext *VideoSearchContext) {
	//顺序执行各个过滤规则
	for _, filter := range searcher.Filters {
		filter.Apply(searchContext)
	}
}

func (searcher *VideoSearcher) Search(searchContext *VideoSearchContext) []*demo.BiliVideo {
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
