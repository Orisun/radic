package index_service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Orisun/radic/v2/types"
	"github.com/Orisun/radic/v2/util"
)

const (
	INDEX_SERVICE = "index_service"
)

type IndexServiceWorker struct {
	Indexer *Indexer // 把正排和倒排索引放到一起
	//跟服务注册相关的配置
	hub      *ServiceHub
	selfAddr string
}

// 初始化索引
func (service *IndexServiceWorker) Init(DocNumEstimate int, dbtype int, DataDir string, etcdServers []string, servicePort int) error {
	service.Indexer = new(Indexer)
	service.Indexer.Init(DocNumEstimate, dbtype, DataDir)

	// 向注册中心注册自己
	if len(etcdServers) > 0 {
		if servicePort <= 1024 {
			return fmt.Errorf("invalid listen port %d, should more than 1024", servicePort)
		}
		selfLocalIp, err := util.GetLocalIP()
		if err != nil {
			panic(err)
		}
		selfLocalIp = "127.0.0.1" //TODO 单机模拟分布式时，把selfLocalIp写死为127.0.0.1
		service.selfAddr = selfLocalIp + ":" + strconv.Itoa(servicePort)
		var heartBeat int64 = 3 //每隔3秒上报一次心跳
		hub := GetServiceHub(etcdServers, heartBeat)
		leaseId, err := hub.Regist(INDEX_SERVICE, service.selfAddr, 0)
		if err != nil {
			panic(err)
		}
		service.hub = hub
		//周期性地注册自己（上报心跳）
		go func() {
			for {
				hub.Regist(INDEX_SERVICE, service.selfAddr, leaseId)
				time.Sleep(time.Duration(heartBeat)*time.Second - 100*time.Millisecond)
			}
		}()
	}

	return nil
}

// 关闭索引
func (service *IndexServiceWorker) Close() error {
	if service.hub != nil {
		service.hub.UnRegist(INDEX_SERVICE, service.selfAddr)
	}
	return service.Indexer.Close()
}

// 从索引上删除文档
func (service *IndexServiceWorker) DeleteDoc(ctx context.Context, docId *DocId) (*AffectedCount, error) {
	return &AffectedCount{int32(service.Indexer.DeleteDoc(docId.DocId))}, nil
}

// 向索引中添加文档(如果已存在，会先删除)
func (service *IndexServiceWorker) AddDoc(ctx context.Context, doc *types.Document) (*AffectedCount, error) {
	n, err := service.Indexer.AddDoc(*doc)
	return &AffectedCount{int32(n)}, err
}

// 系统重启时，直接从索引文件里加载数据
func (service *IndexServiceWorker) LoadFromIndexFile() int {
	return service.Indexer.LoadFromIndexFile()
}

// 检索，返回文档列表
func (service *IndexServiceWorker) Search(ctx context.Context, request *SearchRequest) (*SearchResult, error) {
	result := service.Indexer.Search(request.Query, request.OnFlag, request.OffFlag, request.OrFlags)
	return &SearchResult{Results: result}, nil
}
