package pkg

import "net/rpc"

// HelloServiceName 服务名称
const HelloServiceName = "examples/pkg.HelloService"

// HelloServiceInterface  定义服务接口
type HelloServiceInterface = interface {
	Hello(request string, reply *string) error
}

// RegisterHelloService 注册服务
func RegisterHelloService(svc HelloServiceInterface) error {
	return rpc.RegisterName(HelloServiceName, svc)
}
