package index_service

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/Orisun/radic/v2/util"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	etcdv3 "go.etcd.io/etcd/client/v3"
)

const (
	SERVICE_ROOT_PATH = "/radic/index" //etcd key的前缀
)

// 服务注册中心
type ServiceHub struct {
	client             *etcdv3.Client
	heartbeatFrequency int64 //server每隔几秒钟不停地向中心上报一次心跳（其实就是续一次租约）
	watched            sync.Map
	loadBalancer       LoadBalancer //策略模式。完成同一个任务可以有多种不同的实现方案
}

var (
	serviceHub *ServiceHub //该全局变量包外不可见，包外想使用时通过GetServiceHub()获得
	hubOnce    sync.Once   //单例模式需要用到一个once
)

// ServiceHub的构造函数，单例模式
func GetServiceHub(etcdServers []string, heartbeatFrequency int64) *ServiceHub {
	hubOnce.Do(func() {
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
				loadBalancer:       &RoundRobin{},
			}
		}
	})
	return serviceHub
}

// 注册服务。 第一次注册向etcd写一个key，后续注册仅仅是在续约
//
// service 微服务的名称
//
// endpoint 微服务server的地址
//
// leaseID 租约ID,第一次注册时置为0即可
func (hub *ServiceHub) Regist(service string, endpoint string, leaseID etcdv3.LeaseID) (etcdv3.LeaseID, error) {
	ctx := context.Background()
	if leaseID <= 0 {
		// 创建一个租约，有效期为heartbeatFrequency秒
		if lease, err := hub.client.Grant(ctx, hub.heartbeatFrequency); err != nil {
			util.Log.Printf("创建租约失败：%v", err)
			return 0, err
		} else {
			key := strings.TrimRight(SERVICE_ROOT_PATH, "/") + "/" + service + "/" + endpoint
			// 服务注册
			if _, err = hub.client.Put(ctx, key, "", etcdv3.WithLease(lease.ID)); err != nil { //只需要key，不需要value
				util.Log.Printf("写入服务%s对应的节点%s失败：%v", service, endpoint, err)
				return lease.ID, err
			} else {
				return lease.ID, nil
			}
		}
	} else {
		//续租
		if _, err := hub.client.KeepAliveOnce(ctx, leaseID); err == rpctypes.ErrLeaseNotFound { //续约一次，到期后还得再续约
			return hub.Regist(service, endpoint, 0) //找不到租约，走注册流程(把leaseID置为0)
		} else if err != nil {
			util.Log.Printf("续约失败:%v", err)
			return 0, err
		} else {
			// util.Log.Printf("服务%s对应的节点%s续约成功", service, endpoint)
			return leaseID, nil
		}
	}
}

// 注销服务
func (hub *ServiceHub) UnRegist(service string, endpoint string) error {
	ctx := context.Background()
	key := strings.TrimRight(SERVICE_ROOT_PATH, "/") + "/" + service + "/" + endpoint
	if _, err := hub.client.Delete(ctx, key); err != nil {
		util.Log.Printf("注销服务%s对应的节点%s失败: %v", service, endpoint, err)
		return err
	} else {
		util.Log.Printf("注销服务%s对应的节点%s", service, endpoint)
		return nil
	}
}

// 服务发现。client每次进行RPC调用之前都查询etcd，获取server集合，然后采用负载均衡算法选择一台server。或者也可以把负载均衡的功能放到注册中心，即放到getServiceEndpoints函数里，让它只返回一个server
func (hub *ServiceHub) GetServiceEndpoints(service string) []string {
	ctx := context.Background()
	prefix := strings.TrimRight(SERVICE_ROOT_PATH, "/") + "/" + service + "/"
	if resp, err := hub.client.Get(ctx, prefix, etcdv3.WithPrefix()); err != nil { //按前缀获取key-value
		util.Log.Printf("获取服务%s的节点失败: %v", service, err)
		return nil
	} else {
		endpoints := make([]string, 0, len(resp.Kvs))
		for _, kv := range resp.Kvs {
			path := strings.Split(string(kv.Key), "/") //只需要key，不需要value
			// fmt.Println(string(kv.Key), path[len(path)-1])
			endpoints = append(endpoints, path[len(path)-1])
		}
		util.Log.Printf("刷新%s服务对应的server -- %v\n", service, endpoints)
		return endpoints
	}
}

// 根据负载均衡策略，从众多endpoint里选择一个
func (hub *ServiceHub) GetServiceEndpoint(service string) string {
	return hub.loadBalancer.Take(hub.GetServiceEndpoints(service))
}

// 关闭etcd client connection
func (hub *ServiceHub) Close() {
	hub.client.Close()
}
