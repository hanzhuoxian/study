package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"log"
	"math"
	"net"
	"os"

	"github.com/hanzhuoxian/study/go/examples/grpc/normaltls/data"
	"github.com/hanzhuoxian/study/go/examples/grpc/normaltls/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type circularService struct {
	*model.UnimplementedCircularServiceServer
}

func (c *circularService) Area(ctx context.Context, request *model.AreaRequest) (*model.AreaResponse, error) {
	rjson, err := json.Marshal(request)
	if err != nil {
		log.Printf("request json encode failed: %v", err)
	}
	log.Printf("%s: %s", request.RequestId, string(rjson))

	return &model.AreaResponse{
		Code: 200,
		Area: request.Circular.GetRadius() * request.Circular.GetRadius() * math.Pi,
	}, nil
}
func main() {
	cert, err := tls.LoadX509KeyPair(data.Path("x509/server_cert.pem"), data.Path("x509/server_key.pem"))
	if err != nil {
		log.Fatal(err)
	}
	certPool := x509.NewCertPool()

	ca, err := os.ReadFile(data.Path("x509/ca_cert.pem"))
	if err != nil {
		log.Fatal(err)
	}

	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		log.Fatal("failed to append certs")
	}

	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	})

	grpcServer := grpc.NewServer(grpc.Creds(creds))

	model.RegisterCircularServiceServer(grpcServer, &circularService{})

	listen, err := net.Listen("tcp", "localhost:65530")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Println("server started on localhost:65530")
	if err := grpcServer.Serve(listen); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
