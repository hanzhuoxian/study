package main

import (
	"io"
	"log"
	"math"
	"net"
	"time"

	"github.com/hanzhuoxian/study/go/examples/grpc/stream/model"
	"google.golang.org/grpc"
)

type circularService struct {
	model.UnimplementedCircularServiceServer
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
		log.Printf("recv: %v", r)
		radius := r.Circular.Radius
		if err = stream.Send(&model.AreaResponse{
			Code: 200,
			Area: radius * radius * math.Pi,
		}); err != nil {
			return err
		}
	}
}

func main() {
	lis, err := net.Listen("tcp", ":10000")
	if err != nil {
		log.Fatalf("listen err: %v", err)
	}

	grpcServer := grpc.NewServer(grpc.ConnectionTimeout(5 * time.Second))
	model.RegisterCircularServiceServer(grpcServer, &circularService{})
	log.Println("server listening on :10000")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("serve err: %v", err)
	}
}
