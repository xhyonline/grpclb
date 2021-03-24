// 服务端,服务注册
package etcdv3

import (
	"context"
	"go.etcd.io/etcd/clientv3"
	"log"
	"time"
)

const schema = "grpc-load-balance"

// ServiceRegister 服务注册的实例
type ServiceRegister struct {
	// etcd 客户端
	cli *clientv3.Client
	// 租约 ID
	leaseID clientv3.LeaseID
	// 租约 keepalive 对应的管道
	keepAliveChan <-chan *clientv3.LeaseKeepAliveResponse
	// 服务在 etcd 中注册的 key
	key string
	// 服务在 etcd 中注册的值,其实就是个地址
	val string
}

// NewServiceRegister 注册服务
// 参数释义: endpoints : etcd 连接地址 、 name : 应用程序服务名称 、 addr : 服务启动的地址 、 lease : 持续续租时间 (秒)
func NewServiceRegister(endpoints []string, name, addr string, lease int64) (*ServiceRegister, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	s := &ServiceRegister{
		cli: cli,
		key: "/" + schema + "/" + name + "/" + addr,
		val: addr,
	}
	err = s.putKeyWithLease(lease)
	if err != nil {
		return nil, err
	}
	return s, err
}

// putKeyWithLease 向 etcd 注册,并且 keepalive 续租
func (s *ServiceRegister) putKeyWithLease(lease int64) error {
	// 生成租约
	resp, err := s.cli.Grant(context.Background(), lease)
	if err != nil {
		return err
	}
	// 写入 key 并绑定租约
	_, err = s.cli.Put(context.Background(), s.key, s.val, clientv3.WithLease(resp.ID))
	if err != nil {
		return err
	}
	// 租约 keepalive
	leaseChan, err := s.cli.KeepAlive(context.Background(), resp.ID)
	if err != nil {
		return err
	}
	s.leaseID = resp.ID
	s.keepAliveChan = leaseChan
	log.Println("服务端注册成功 key:" + s.key + " value:" + s.val)
	return nil
}

// ListenLeaseRespChan 监听续租情况
func (s *ServiceRegister) ListenLeaseRespChan() {
	for leaseKeepResp := range s.keepAliveChan {
		log.Println("续约成功", leaseKeepResp)
	}
	log.Println("关闭续租")
}

// Close 关闭服务,撤销租约
func (s *ServiceRegister) Close() error {
	//撤销租约
	if _, err := s.cli.Revoke(context.Background(), s.leaseID); err != nil {
		return err
	}
	log.Println("撤销租约")
	return s.cli.Close()
}

// TODO 还需要监听移除退出,例如 CTRL+C 退出,然后让他执行 Close(),及时退出续租
