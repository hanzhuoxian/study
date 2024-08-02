package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/daymenu/gostudy/examples/grpc/stream/model"
	"google.golang.org/grpc"
)

func main() {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithBlock())
	conn, err := grpc.Dial("localhost:10000", opts...)
	if err != nil {
		log.Fatalf("dial err : %v", err)
	}

	defer conn.Close()

	client := model.NewCircularServiceClient(conn)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ca, err := client.Area(timeoutCtx)
	if err != nil {
		log.Fatal("create area client err", err)
	}
	waitc := make(chan struct{})
	go func() {
		for {
			r, err := ca.Recv()
			if err == io.EOF {
				close(waitc)
			}
			if err != nil {
				log.Fatal("go recv is die")
			}
			fmt.Println("c receive r is :", r)
		}
	}()
	req := &model.AreaRequest{Circular: &model.Circular{Dot: &model.Point{X: 1, Y: 1}, Radius: 3}}
	ca.Send(req)
	ca.Send(req)
	ca.CloseSend()
	<-waitc
}
