package main

import (
	"fmt"
	"github.com/xhyonline/grpclb/etcdv3"
	"github.com/xhyonline/grpclb/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
	"log"
	"time"
	"context"
)

func main() {
	var etcdEndpoints = []string{
		"121.5.62.93:2379",
	}
	r:=etcdv3.NewServiceDiscovery(etcdEndpoints)
	resolver.Register(r)
	// 连接服务器
	// r.Scheme()+"://8.8.8.8/user-center" 中,至关重要的就是 user-center,这是你的服务名称
	// 虽然在这里没有指定连接的端口号和 IP ,你可能比较奇怪,它是怎么连接上的,这一切都要归功于 Build 方法去 etcd 中发现 IP ,
	// 并且加入到了自己的负载均衡器中,详细请点开 etcdv3.NewServiceDiscovery 方法查看
	conn, err := grpc.Dial(r.Scheme()+":///user-center",
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
		grpc.WithInsecure())
	if err!=nil {
		log.Fatalf("客户端连接失败 %s",err)
	}
	defer conn.Close()
	// 与 proto 绑定
	grpcClient := pb.NewSimpleClient(conn)

	// 连续请求 100 次 GRPC 服务,观察它的负载均衡
	for i:=0;i<100;i++ {
		resp, err := grpcClient.Hello(context.Background(), &pb.Empty{})
		if err != nil {
			panic(err)
		}
		fmt.Println(resp.Message)
		time.Sleep(time.Second)
	}
}
