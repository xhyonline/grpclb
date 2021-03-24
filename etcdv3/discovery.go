// 服务发现
package etcdv3

import (
	"context"
	"go.etcd.io/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"google.golang.org/grpc/resolver"
	"log"
	"sync"
	"time"
)

// ServiceDiscovery 客户端服务发现
type ServiceDiscovery struct {
	// etcd 客户端
	cli *clientv3.Client
	// 负载均衡器
	cc resolver.ClientConn
	// 服务列表
	serverList map[string]resolver.Address
	// 服务列表锁
	lock sync.Mutex
}

// Build 实现了第三方方法 resolver.Register() 的入参接口 resolver.Builder
// 当客户端使用 grpc.Dial 时,将会自动触发该函数,有点像一个 hook 钩子
// 参数释义:
// target : 当客户端调用 grpc.Dial() 方法时,会将入参解析到 target 中,例如 grpc.Dial("dns://some_authority/foo.bar") 就会解析成  &Target{Scheme: "dns", Authority: "some_authority", Endpoint: "foo.bar"}
// cc 负载均衡器
func (s *ServiceDiscovery) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	log.Println("grpc 启动触发")
	s.cc = cc
	s.serverList = make(map[string]resolver.Address)
	prefix := "/" + target.Scheme + "/" + target.Endpoint + "/"
	log.Println("开始获取" + prefix + "下所有的key")

	// 获取前缀下现有的 key
	resp, err := s.cli.Get(context.Background(), prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	// 获取完毕,并且注册服务,加入负载均衡器
	for _, ev := range resp.Kvs {
		s.SetServiceList(string(ev.Key),string(ev.Value))
	}
	// 注册完毕,开始监视前缀,变更修改的 server
	//监视前缀，修改变更的server
	go s.watcher(prefix)
	return s, nil
}

// Scheme 实现了第三方方法 resolver.Register() 的入参接口 resolver.Builder
func (s *ServiceDiscovery) Scheme() string{
	return schema
}

// ResolveNow 实现第三方 resolver.Resolver 的接口,监视目标更新
func (s *ServiceDiscovery) ResolveNow(rn resolver.ResolveNowOptions) {
	log.Println("ResolveNow")
}

// Close 实现 resolver.Resolver 的关闭接口
func (s *ServiceDiscovery) Close() {
	log.Println("Close")
	s.cli.Close()
}

// SetServiceList 设置服务
func (s *ServiceDiscovery) SetServiceList(key, val string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	// 写入服务
	s.serverList[key] = resolver.Address{Addr: val}
	// 写入完毕,将改地址加入负载均衡器
	s.cc.UpdateState(resolver.State{
		Addresses: s.GetServices(),
	})
	log.Println("新增服务地址:" + val + " 并已经加入负载均衡器")
}

// DelServiceList 删除服务地址
func (s *ServiceDiscovery) DelServiceList(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	addr := s.serverList[key]
	delete(s.serverList, key)
	// 删除完毕,更新负载均衡器
	s.cc.UpdateState(resolver.State{
		Addresses: s.GetServices(),
	})
	log.Println("删除服务地址:" + addr.Addr + " 并移除负载均衡器")
}

// GetServices 获取当前的服务列表
func (s *ServiceDiscovery) GetServices() []resolver.Address {
	address := make([]resolver.Address, 0, len(s.serverList))
	for _, v := range s.serverList {
		address = append(address, v)
	}
	return address
}

// watcher 监控
func (s *ServiceDiscovery) watcher(prefix string){
	ch:=s.cli.Watch(context.Background(),prefix,clientv3.WithPrefix())
	log.Println("开始监控前缀为"+prefix+"的变化")
	for resp:=range ch{
		for _,ev:=range resp.Events{
			switch ev.Type {
			case mvccpb.PUT:	// 如果是新增或者修改
				s.SetServiceList(string(ev.Kv.Key), string(ev.Kv.Value))
			case mvccpb.DELETE: //删除
				s.DelServiceList(string(ev.Kv.Key))
			}
		}
	}
}

// NewServiceDiscovery 实例化一个服务发现
func NewServiceDiscovery(endpoints []string) resolver.Builder {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	return &ServiceDiscovery{
		cli: cli,
	}
}
