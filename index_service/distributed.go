package index_service

import (
	context "context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	types "github.com/Orisun/radic/v2/types"
	"github.com/Orisun/radic/v2/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

type Sentinel struct {
	hub      IServiceHub // 从Hub上获取IndexServiceWorker集合。可能是直接访问ServiceHub，也可能是走代理
	connPool sync.Map    // 与各个IndexServiceWorker建立的连接。把连接缓存起来，避免每次都重建连接
}

func NewSentinel(etcdServers []string) *Sentinel {
	return &Sentinel{
		hub: GetServiceHub(etcdServers, 3), //直接访问ServiceHub
		// hub: GetServiceHubProxy(etcdServers, 3, 100),  //走代理
		connPool: sync.Map{},
	}
}

func (sentinel *Sentinel) GetGrpcConn(endpoint string) *grpc.ClientConn {
	if v, exists := sentinel.connPool.Load(endpoint); exists {
		conn := v.(*grpc.ClientConn)
		//如果连接状态不可用，则从连接缓存中删除
		if conn.GetState() == connectivity.TransientFailure || conn.GetState() == connectivity.Shutdown {
			conn.Close()
			sentinel.connPool.Delete(endpoint)
		} else {
			return conn //缓存中有该连接，则直接返回
		}
	}
	//连接到服务端
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond) //控制连接超时
	defer cancel()
	conn, err := grpc.DialContext(
		ctx,
		endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()), //Credential即使为空，也必须设置
		//grpc.Dial是异步连接的，连接状态为正在连接。但如果你设置了 grpc.WithBlock 选项，就会阻塞等待（等待握手成功）。另外你需要注意，当未设置 grpc.WithBlock 时，ctx 超时控制对其无任何效果。
		grpc.WithBlock(),
	)
	if err != nil {
		util.Log.Printf("dial %s failed: %s", endpoint, err)
		return nil
	}
	util.Log.Printf("connect to grpc server %s", endpoint)
	sentinel.connPool.Store(endpoint, conn)
	return conn
}

// 向集群中添加文档(如果已存在，会先删除)
func (sentinel *Sentinel) AddDoc(doc types.Document) (int, error) {
	endpoints := sentinel.hub.GetServiceEndpoints(INDEX_SERVICE)
	if len(endpoints) == 0 {
		return 0, fmt.Errorf("no index worker")
	}
	idx := rand.Intn(len(endpoints)) // 随机选择一台index worker，把doc添加到它上面去
	endpoint := endpoints[idx]
	conn := sentinel.GetGrpcConn(endpoint)
	if conn == nil {
		return 0, fmt.Errorf("connect to worker %s failed", endpoint)
	}
	client := NewIndexServiceClient(conn)
	affected, err := client.AddDoc(context.Background(), &doc)
	if err != nil {
		return 0, err
	}
	util.Log.Printf("add %d doc to worker %s", affected.Count, endpoint)
	return int(affected.Count), nil
}

// 从集群上删除docId，返回成功删除的doc数（正常情况下不会超过1）
func (sentinel *Sentinel) DeleteDoc(docId string) int {
	endpoints := sentinel.hub.GetServiceEndpoints(INDEX_SERVICE)
	if len(endpoints) == 0 {
		return 0
	}
	var n int32
	wg := sync.WaitGroup{}
	wg.Add(len(endpoints))
	for _, endpoint := range endpoints {
		go func(endpoint string) { //并行到各个IndexServiceWorker上把docId删除。正常情况下只有一个worker上有该doc
			defer wg.Done()
			conn := sentinel.GetGrpcConn(endpoint)
			if conn != nil {
				client := NewIndexServiceClient(conn)
				affected, err := client.DeleteDoc(context.Background(), &DocId{docId})
				if err != nil {
					util.Log.Printf("delete doc %s from worker %s failed: %s", docId, endpoint, err)
				} else {
					if affected.Count > 0 {
						atomic.AddInt32(&n, affected.Count)
						util.Log.Printf("delete %d from worker %s", affected.Count, endpoint)
					}
				}
			}
		}(endpoint)
	}
	wg.Wait()
	return int(atomic.LoadInt32(&n))
}

func (sentinel *Sentinel) Search(query *types.TermQuery, onFlag uint64, offFlag uint64, orFlags []uint64) []*types.Document {
	endpoints := sentinel.hub.GetServiceEndpoints(INDEX_SERVICE)
	if len(endpoints) == 0 {
		return nil
	}
	docs := make([]*types.Document, 0, 1000)
	resultCh := make(chan *types.Document, 1000)
	wg := sync.WaitGroup{}
	wg.Add(len(endpoints))
	for _, endpoint := range endpoints {
		go func(endpoint string) {
			defer wg.Done()
			conn := sentinel.GetGrpcConn(endpoint)
			if conn != nil {
				client := NewIndexServiceClient(conn)
				result, err := client.Search(context.Background(), &SearchRequest{Query: query, OnFlag: onFlag, OffFlag: offFlag, OrFlags: orFlags})
				if err != nil {
					util.Log.Printf("search from cluster failed: %s", err)
				} else {
					if len(result.Results) > 0 {
						util.Log.Printf("search %d doc from worker %s", len(result.Results), endpoint)
						for _, doc := range result.Results {
							resultCh <- doc
						}
					}
				}
			}
		}(endpoint)
	}

	receiveFinish := make(chan struct{})
	go func() { //为什么要放到一个子协程里？因为里面有个无限for循环，只有“//1”执行了该for循环才能退出
		for {
			doc, ok := <-resultCh
			if !ok {
				break //2
			}
			docs = append(docs, doc)
		}
		receiveFinish <- struct{}{} //3
	}()
	wg.Wait()
	close(resultCh) //1
	<-receiveFinish //4
	return docs
}
