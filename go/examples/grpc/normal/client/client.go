package main

import (
	"context"
	"log"

	"github.com/hanzhuoxian/study/go/examples/grpc/normaltls/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func main() {

	c, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	defer c.Close()

	req := &model.AreaRequest{
		Circular: &model.Circular{
			Dot: &model.Point{
				X: 5,
				Y: 5,
			},
		},
		Color: &wrapperspb.Int64Value{Value: 1},
	}

	grpcClient := model.NewCircularServiceClient(c)
	resp, err := grpcClient.Area(context.Background(), req)
	if err != nil {
		log.Fatalf("failed to call Area: %v", err)
	}
	log.Printf("Area: %f", resp.Area)
}
