package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"time"

	"github.com/daymenu/gostudy/examples/grpc/stream/model"
	"google.golang.org/grpc"
)

type circularService struct {
	*model.UnimplementedCircularServiceServer
}

func (c *circularService) Area(stream model.CircularService_AreaServer) error {
	for {

		r, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Println(r)
		resp := new(model.AreaResponse)
		resp.Code = 200
		radius := r.Circular.Radius
		resp.Area = radius * radius * math.Pi
		err = stream.Send(resp)
		fmt.Println(err)
	}
}

func main() {

	lis, err := net.Listen("tcp", fmt.Sprintf(":10000"))
	if err != nil {
		log.Fatalf("listen err：%v", err)
	}

	var opts []grpc.ServerOption
	opts = append(opts, grpc.ConnectionTimeout(1*time.Second))
	grpcServer := grpc.NewServer(opts...)
	log.Println("register area server")
	model.RegisterCircularServiceServer(grpcServer, new(circularService))
	log.Println("server is serve")
	grpcServer.Serve(lis)
}
