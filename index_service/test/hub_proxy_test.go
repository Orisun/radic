package test

import (
	"fmt"
	"testing"

	"github.com/Orisun/radic/v2/index_service"
)

func TestGetServiceEndpointsByProxy(t *testing.T) {
	proxy := index_service.GetServiceHubProxy(etcdServers, 3, 10)

	endpoint := "127.0.0.1:5000"
	proxy.Regist(serviceName, endpoint, 0)
	defer proxy.UnRegist(serviceName, endpoint)
	endpoints := proxy.GetServiceEndpoints(serviceName)
	fmt.Printf("endpoints %v\n", endpoints)

	endpoint = "127.0.0.2:5000"
	proxy.Regist(serviceName, endpoint, 0)
	defer proxy.UnRegist(serviceName, endpoint)
	endpoints = proxy.GetServiceEndpoints(serviceName)
	fmt.Printf("endpoints %v\n", endpoints)

	endpoint = "127.0.0.3:5000"
	proxy.Regist(serviceName, endpoint, 0)
	defer proxy.UnRegist(serviceName, endpoint)
	endpoints = proxy.GetServiceEndpoints(serviceName)
	fmt.Printf("endpoints %v\n", endpoints)
}

// go test -v ./index_service/test -run=^TestGetServiceEndpointsByProxy$ -count=1
