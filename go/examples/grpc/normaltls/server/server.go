package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net"

	"github.com/daymenu/gostudy/examples/grpc/normaltls/data"
	"github.com/daymenu/gostudy/examples/grpc/normaltls/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type circularService struct {
	*model.UnimplementedCircularServiceServer
}

func (c *circularService) Area(ctx context.Context, request *model.AreaRequest) (*model.AreaResponse, error) {
	resp := new(model.AreaResponse)
	resp.Code = 200
	age := request.GetColor()
	fmt.Println("color:", request.GetColor().GetValue())
	if age != nil {
		log.Println(age)
	}
	resp.Area = request.Circular.GetRadius() * request.Circular.GetRadius() * math.Pi
	rjson, err := json.Marshal(request)
	if err != nil {
		log.Printf("request json encode failed")
	}
	log.Printf("%s:color[%s]", request.RequestId, string(rjson))
	return resp, nil
}
func newServer() *circularService {
	s := &circularService{}
	return s
}
func main() {
	cert, err := tls.LoadX509KeyPair(data.Path("x509/server_cert.pem"), data.Path("x509/server_key.pem"))
	if err != nil {
		log.Fatal(err)
	}
	certPool := x509.NewCertPool()

	ca, err := ioutil.ReadFile(data.Path("x509/ca_cert.pem"))
	if err != nil {
		log.Fatal(err)
	}

	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		log.Fatal("failed to append cserts")
	}

	log.Println("server is start ....")
	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	})

	grpcServer := grpc.NewServer(grpc.Creds(creds))

	model.RegisterCircularServiceServer(grpcServer, newServer())

	listen, err := net.Listen("tcp", "localhost:65530")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer.Serve(listen)
}
