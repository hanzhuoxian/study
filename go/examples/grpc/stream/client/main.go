package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/hanzhuoxian/study/go/examples/grpc/stream/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("localhost:10000",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("dial err: %v", err)
	}
	defer conn.Close()

	client := model.NewCircularServiceClient(conn)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clientArea, err := client.Area(timeoutCtx)
	if err != nil {
		log.Fatalf("create area client err: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		for {
			r, err := clientArea.Recv()
			if err == io.EOF {
				errCh <- nil
				return
			}
			if err != nil {
				errCh <- err
				return
			}
			fmt.Println("received:", r)
		}
	}()

	req := &model.AreaRequest{Circular: &model.Circular{Dot: &model.Point{X: 1, Y: 1}, Radius: 3}}
	for range 2 {
		if err := clientArea.Send(req); err != nil {
			log.Fatalf("send err: %v", err)
		}
	}
	if err := clientArea.CloseSend(); err != nil {
		log.Fatalf("close send err: %v", err)
	}

	if err := <-errCh; err != nil {
		log.Fatalf("recv err: %v", err)
	}
}
