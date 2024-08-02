package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/daymenu/gostudy/examples/grpc/normaltls/data"
	"github.com/daymenu/gostudy/examples/grpc/normaltls/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	// 加载证书
	cert, err := tls.LoadX509KeyPair(data.Path("x509/client_cert.pem"), data.Path("x509/client_key.pem"))
	if err != nil {
		log.Fatal(err)
	}

	// 创建x509
	certPool := x509.NewCertPool()

	ca, err := ioutil.ReadFile(data.Path("x509/ca_cert.pem"))
	if err != nil {
		log.Fatal(err)
	}

	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		log.Fatal("failed to append cserts")
	}

	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ServerName:   "x.daymenu.cn",
		RootCAs:      certPool,
	})

	conn, err := grpc.Dial("localhost:65530", grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	log.Println("connect server success")
	defer conn.Close()

	client := model.NewCircularServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	circularPoint := &model.Point{X: 1.1, Y: 1.1}
	radius := 1.1

	resp, err := client.Area(ctx, &model.AreaRequest{
		RequestId: "2233",
		Circular:  &model.Circular{Dot: circularPoint, Radius: radius},
		// Color:     &wrapperspb.Int64Value{Value: 1},
	})
	if err != nil {
		log.Fatal(err)
	}
	if resp.Code != http.StatusOK {
		fmt.Println(err)
	}
	fmt.Printf("圆点为：(%.2f,%.2f)\n圆的半径为： %.2f\n圆的面积：%.f \n", circularPoint.X, circularPoint.Y, radius, resp.GetArea())
}
