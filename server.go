// server.go 是模拟应用程序服务
// 当它启动时,会向 etcd 注册,并且持续续租,关闭时会关闭销租
package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/xhyonline/grpclb/etcdv3"
	"github.com/xhyonline/grpclb/pb"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
)

var port string

// SimpleService
type SimpleService struct {
}

// Hello 实现 proto 定义的接口
func (s *SimpleService) Hello(ctx context.Context, empty *pb.Empty) (*pb.Response, error) {
	fmt.Println("请求打进 port 为" + port + "的服务了")
	return &pb.Response{
		Message: "服务端:" + port + "接收到了请求",
	}, nil
}

func main() {
	// 服务名称
	const serverName = "user-center"
	// 定义 etcd 地址
	var etcdEndpoints = []string{
		"127.0.0.1:2379",
	}
	flag.StringVar(&port, "port", "", "端口不能不填写")
	flag.Parse()
	if port == "" {
		fmt.Println("服务端端口不能不填写")
		os.Exit(1)
	}
	addr := "127.0.0.1:" + port
	// 监听本地端口
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("监听 tcp 端口错误 err %v", err)
	}
	log.Println("监听 tcp 端口成功,地址:", addr)
	// 新建 grpc 服务
	grpcServer := grpc.NewServer()
	// 在gRPC服务器注册我们的服务
	pb.RegisterSimpleServer(grpcServer, &SimpleService{})
	// 把服务注册到 etcd
	s, err := etcdv3.NewServiceRegister(etcdEndpoints, serverName, addr, 5)
	if err != nil {
		log.Fatalf("服务注册出错: %v", err)
	}
	defer s.Close()
	//用服务器 Serve() 方法以及我们的端口信息区实现阻塞等待，直到进程被杀死或者 Stop() 被调用
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("grpc 服务启动失败: %v", err)
	}
}
