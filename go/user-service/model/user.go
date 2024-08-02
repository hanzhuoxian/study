package model

import (
	pb "github.com/daymenu/user-service/proto/user"
	"github.com/jinzhu/gorm"
)

// User 定义User接口
type User interface {
	GetAll() ([]*pb.User, error)
	Page(req *pb.Request) ([]*pb.User, error)
	Get(id int64) (*pb.User, error)
	Create(user *pb.User) error
	Update(user *pb.User) error
	GetByEmail(email string) (*pb.User, error)
}

// UserModel 结构体
type UserModel struct {
	Db *gorm.DB
}

// GetAll 获取全部数据
func (user *UserModel) GetAll() ([]*pb.User, error) {
	var users []*pb.User
	if err := user.Db.Find(&users).Error; err != nil {
		return users, err
	}
	return users, nil
}

// Page 分页获取用户
func (user *UserModel) Page(req *pb.Request) ([]*pb.User, error) {
	var users []*pb.User
	return users, nil
}

// Get 根据用户Id获取一个用户
func (user *UserModel) Get(id int64) (*pb.User, error) {
	var u *pb.User
	u.Id = id
	if err := user.Db.First(&user).Error; err != nil {
		return nil, err
	}
	return u, nil
}

// GetByEmail 根据邮箱号获取一个用户
func (user *UserModel) GetByEmail(email string) (*pb.User, error) {
	var u *pb.User
	if err := user.Db.Where("email = ?", email).
		First(&user).Error; err != nil {
		return nil, err
	}
	return u, nil
}

// Create 创建一个用户
func (user *UserModel) Create(u *pb.User) error {
	if err := user.Db.Create(u).Error; err != nil {
		return err
	}
	return nil
}

// Update 更新一个用户
func (user *UserModel) Update(u *pb.User) error {
	if err := user.Db.Save(u).Error; err != nil {
		return nil
	}
	return nil
}
