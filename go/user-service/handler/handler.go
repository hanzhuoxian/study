package handler

import (
	"context"
	"fmt"

	"github.com/daymenu/user-service/model"
	"github.com/daymenu/user-service/token"

	pb "github.com/daymenu/user-service/proto/user"
	"github.com/micro/go-micro"
	"golang.org/x/crypto/bcrypt"
)

const topic = "user.created"

type Service struct {
	User         model.User
	TokenService token.Authable
	Publisher    micro.Publisher
}

func (srv *Service) Get(ctx context.Context, req *pb.User, res *pb.Response) error {
	user, err := srv.User.Get(req.Id)
	if err != nil {
		return err
	}
	res.User = user
	return nil
}

func (srv *Service) Create(ctx context.Context, user *pb.User, res *pb.Response) error {
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("error hashing password:%v", err)
	}
	user.Password = string(hashedPass)
	err = srv.User.Create(user)
	if err != nil {
		return err
	}
	res.User = user
	if err := srv.Publisher.Publish(ctx, user); err != nil {
		return err
	}
	return nil
}

func (srv *Service) Update(ctx context.Context, user *pb.User, res *pb.Response) error {
	// password is not empty, change password
	if user.Password != "" {
		hashedPass, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("error hashing password:%v", err)
		}
		user.Password = string(hashedPass)

	}
	err := srv.User.Update(user)
	if err != nil {
		return err
	}
	res.User = user
	return nil
}

func (srv *Service) Page(ctx context.Context, req *pb.Request, res *pb.Response) error {
	users, err := srv.User.Page(req)
	if err != nil {
		return err
	}
	res.Users = users
	return nil
}

func (srv *Service) GetAll(ctx context.Context, req *pb.Request, res *pb.Response) error {
	users, err := srv.User.GetAll()
	if err != nil {
		return err
	}
	res.Users = users
	return nil
}

func (srv *Service) Login(ctx context.Context, user *pb.User, res *pb.Token) error {
	user, err := srv.User.GetByEmail(user.Email)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(user.Password)); err != nil {
		return err
	}
	token, err := srv.TokenService.Encode(user)
	if err != nil {
		return err
	}
	res.Token = token
	return nil
}

func (srv *Service) ValidateToken(ctx context.Context, req *pb.Token, res *pb.Token) error {
	claims, err := srv.TokenService.Decode(req.Token)
	if err != nil {
		return err
	}
	if claims.User.Id == 0 {
		return fmt.Errorf("invalid user")
	}
	res.Valid = true
	return nil
}
