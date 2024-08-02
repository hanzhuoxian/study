package main

import (
	"log"

	"github.com/daymenu/user-service/handler"
	"github.com/daymenu/user-service/token"

	"github.com/daymenu/user-service/model"

	pb "github.com/daymenu/user-service/proto/user"
	"github.com/micro/go-micro"
)

func main() {
	db, err := model.CreateConn()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	db.AutoMigrate(&pb.User{})

	user := &model.UserModel{Db: db}
	tokenService := &token.TokenService{User: user}
	srv := micro.NewService(
		micro.Name("daymenu.user"),
		micro.Version("latest"),
	)

	publisher := micro.NewPublisher("user.created", srv.Client())
	srv.Init()
	pb.RegisterUserServiceHandler(srv.Server(), &handler.Service{
		User:         user,
		TokenService: tokenService,
		Publisher:    publisher},
	)
	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}
