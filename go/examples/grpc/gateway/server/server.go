package main

import (
	"context"
	"log"
	"net"

	"github.com/hanzhuoxian/study/go/examples/grpc/normaltls/model"
	"google.golang.org/grpc"
)

type circularService struct {
	model.UnimplementedCircularServiceServer
}

func (s *circularService) Area(ctx context.Context, request *model.AreaRequest) (*model.AreaResponse, error) {
	area := 3.14 * request.Circular.Dot.X * request.Circular.Dot.Y
	return &model.AreaResponse{Area: area}, nil
}

func main() {
	// 监听端口
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	// 创建 gRPC 服务器
	s := grpc.NewServer()
	// 注册服务
	model.RegisterCircularServiceServer(s, &circularService{})
	log.Printf("Server is listening on port 50051...")
	// 启动服务器
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
