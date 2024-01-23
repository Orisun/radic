package index_service

import (
	context "context"
	"strings"
	"sync"
	"time"

	"github.com/Orisun/radic/v2/util"
	etcdv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/time/rate"
)

type IServiceHub interface {
	Regist(service string, endpoint string, leaseID etcdv3.LeaseID) (etcdv3.LeaseID, error) // 注册服务
	UnRegist(service string, endpoint string) error                                         // 注销服务
	GetServiceEndpoints(service string) []string                                            //服务发现
}

// 代理模式。对ServiceHub做一层代理，想访问endpoints时需要通过代理，代理提供了2个功能：缓存和限流保护
type HubProxy struct {
	hub           *ServiceHub
	endpointCache sync.Map //维护每一个service下的所有servers
	limiter       *rate.Limiter
}

var (
	proxy     *HubProxy
	proxyOnce sync.Once
)

// HubProxy的构造函数，单例模式。
//
// qps一秒钟最多允许请求多少次
func GetServiceHubProxy(etcdServers []string, heartbeatFrequency int64, qps int) *HubProxy {
	proxyOnce.Do(func() {
		if proxy == nil {
			if client, err := etcdv3.New(
				etcdv3.Config{
					Endpoints:   etcdServers,
					DialTimeout: 3 * time.Second,
				},
			); err != nil {
				util.Log.Fatalf("连接不上etcd服务器: %v", err) //发生log.Fatal时go进程会直接退出
			} else {
				serviceHub = &ServiceHub{
					client:             client,
					heartbeatFrequency: heartbeatFrequency, //租约的有效期
				}
				proxy = &HubProxy{
					hub:           serviceHub,
					endpointCache: sync.Map{},
					limiter:       rate.NewLimiter(rate.Every(time.Duration(1e9/qps)*time.Nanosecond), 1), //每隔1E9/qps纳秒产生一个令牌，即一秒钟之内产生qps个令牌
				}
			}
		}
	})
	return proxy
}

// 注册服务
func (proxy *HubProxy) Regist(service string, endpoint string, leaseID etcdv3.LeaseID) (etcdv3.LeaseID, error) {
	return proxy.hub.Regist(service, endpoint, leaseID)
}

// 注销服务
func (proxy *HubProxy) UnRegist(service string, endpoint string) error {
	return proxy.hub.UnRegist(service, endpoint)
}

func (proxy *HubProxy) watchEndpointsOfService(service string) {
	if _, exists := proxy.hub.watched.LoadOrStore(service, true); exists {
		return //监听过了，不用重复监听
	}
	ctx := context.Background()
	prefix := strings.TrimRight(SERVICE_ROOT_PATH, "/") + "/" + service + "/"
	ch := proxy.hub.client.Watch(ctx, prefix, etcdv3.WithPrefix()) //根据前缀监听，每一个修改都会放入管道ch
	util.Log.Printf("监听服务%s的节点变化", service)
	go func() {
		for response := range ch { //遍历管道。这是个死循环，除非关闭管道
			for _, event := range response.Events { //每次从ch里取出来的是事件的集合
				path := strings.Split(string(event.Kv.Key), "/")
				if len(path) > 2 {
					service := path[len(path)-2]
					// 跟etcd进行一次全量同步
					endpoints := proxy.hub.GetServiceEndpoints(service)
					if len(endpoints) > 0 {
						proxy.endpointCache.Store(service, endpoints) //查询etcd的结果放入本地缓存
					} else {
						proxy.endpointCache.Delete(service) //该service下已经没有endpoint
					}
				}
			}
		}
	}()
}

// 服务发现
//
// 把第一次查询etcd的结果缓存起来，然后安装一个Watcher，仅etcd数据变化时更新本地缓存，这样可以降低etcd的访问压力
//
// 同时加上限流保护
func (proxy *HubProxy) GetServiceEndpoints(service string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	proxy.limiter.Wait(ctx) //阻塞，直到桶中有1个令牌或超时。

	// if !proxy.limiter.Allow(time.Now()) { //不阻塞，如果桶中没有1个令牌，则函数直接返回空，即没有可用的endpoints
	// 	return nil
	// }

	proxy.watchEndpointsOfService(service) //监听etcd的数据变化，及时更新本地缓存
	if endpoints, exists := proxy.endpointCache.Load(service); exists {
		return endpoints.([]string)
	} else {
		endpoints := proxy.hub.GetServiceEndpoints(service)
		if len(endpoints) > 0 {
			proxy.endpointCache.Store(service, endpoints) //查询etcd的结果放入本地缓存
		}
		return endpoints
	}
}
