package main

import (
	"log"
	"net"
	"net/rpc"
)

// HelloService hello service struct
type HelloService struct{}

// Hello rpc hello
func (h *HelloService) Hello(request string, reply *string) error {
	*reply = "hello, " + request
	return nil
}

// Ad hello rpc ad
func (h *HelloService) Ad(request string, ad *string) error {
	*ad = "性感荷官在线发牌！！"
	return nil
}
func main() {
	rpc.RegisterName("HelloService", new(HelloService))

	listener, err := net.Listen("tcp", ":1234")
	if err != nil {
		log.Fatal("ListenTCP error:", err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal("Accept error:", err)
		}
		rpc.ServeConn(conn)
	}
}
