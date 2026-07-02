package main

import (
	"encoding/json"
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 用户表 权限表 关联表
// User 用户表
type User struct {
	gorm.Model
	Name  string
	Auths []*Auth `gorm:"many2many:user_auth;"`
}

// Auth 权限表
type Auth struct {
	gorm.Model
	Name  string
	Users []*User `gorm:"many2many:user_auth;"`
}

// UserAuth 自定义关联表
type UserAuth struct {
	UserID    int            `gorm:"primaryKey"`
	AuthID    int            `gorm:"primaryKey"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Status    int
}

func main() {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		panic("failed to connect database")
	}

	db.Debug()
	// 修改 Person 的 Addresses 字段的连接表为 PersonAddress
	// PersonAddress 必须定义好所需的外键，否则会报错
	if err = db.SetupJoinTable(&User{}, "Auths", &UserAuth{}); err != nil {
		panic("failed to connect database")
	}

	if err = db.SetupJoinTable(&Auth{}, "Users", &UserAuth{}); err != nil {
		panic("failed to connect database")
	}

	// 迁移 schema
	db.AutoMigrate(&User{})
	db.AutoMigrate(&Auth{})
	db.AutoMigrate(&UserAuth{})
	db.AutoMigrate(&Person{}, &Notes{})
	// err = db.Create(&Person{Name: "zhangsan", Notes: []*Notes{{Notes: "vip"}, {Notes: "buyer"}}}).Error
	if err != nil {
		panic(err)
	}
	var persons []Person
	err = db.Model(&Person{}).Preload("Notes").Find(&persons).Error
	if err != nil {
		panic(err)
	}

	by, _ := json.MarshalIndent(persons, "", " ")
	fmt.Printf("%s", string(by))

	// Create
	// db.Create(&User{Name: "zhangsan", Auths: []*Auth{{Name: "admin"}, {Name: "edit"}}})
	// var auths []Auth
	// err = db.Model(&Auth{}).Preload("Users").Find(&auths).Error
	// if err != nil {
	// 	panic("select failed")
	// }
	// by, _ := json.MarshalIndent(auths, "", " ")
	// fmt.Printf("%s", string(by))

}
