# GRPC + ETCD  服务发现与负载均衡示例



采用的是,应用程序进程内负载均衡的模式

启动命令:

etcd 请自行在 client.go 与 server.go 中修改为自己的地址,并且启动

服务端

```
go run server.go --port 8000
go run server.go --port 8001
go run server.go --port 8002
```

客户端

```
go run client.go
```

并且客户端与服务端运行时刻。如果服务端在 run 一个端口,客户端能及时将请求均摊