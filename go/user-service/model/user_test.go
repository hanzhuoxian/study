package model

import (
	"testing"

	pb "github.com/daymenu/user-service/proto/user"
)

func TestCreate(t *testing.T) {
	db, err := CreateConn()
	if err != nil {
		t.Fail()
	}
	userModel := &UserModel{db}
	user := pb.User{
		Name: "hj",
	}
	if err := userModel.Create(&user); err != nil {
		t.Fail()
	}

}
