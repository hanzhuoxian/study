// Package user 存放用户 Model 相关逻辑
package user

import (
	"gohub/app/models"
	"gohub/pkg/database"
	"gohub/pkg/hash"

	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	models.BaseModel

	Name         string `json:"name,omitempty"`
	City         string `json:"city,omitempty"`
	Introduction string `json:"introduction,omitempty"`
	Avatar       string `json:"avatar,omitempty"`
	Email        string `json:"-"`
	Phone        string `json:"-"`
	Password     string `json:"-"`

	models.CommonTimestampsFields
}

// Create 创建用户
func (u *User) Create() {
	database.DB.Create(&u)
}

func (u *User) Save() int64 {
	tx := database.DB.Save(&u)
	return tx.RowsAffected
}

// BeforeSave GORM 的模型钩子， 在创建和更新模型前调用
func (u *User) BeforeSave(tx *gorm.DB) (err error) {
	if !hash.BcryptIsHash(u.Password) {
		u.Password = hash.BcryptHash(u.Password)
	}
	return
}

// ComparePassword 校验密码是否正确
func (u *User) ComparePassword(passowrd string) bool {
	return hash.BcryptCheck(passowrd, u.Password)
}

// GetByPhone 通过手机号来获取用户
func GetByPhone(phone string) (userModel User) {
	database.DB.Where("phone = ?", phone).First(&userModel)
	return
}

// GetByEmail 通过邮箱来获取用户
func GetByEmail(email string) (userModel User) {
	database.DB.Where("email = ?", email).First(&userModel)
	return
}

// GetByMulti 通过用户名、手机号、邮箱登录
func GetByMulti(loginID string) (userModel User) {
	database.DB.Where("phone = ?", loginID).Or("email = ?", loginID).Or("name = ?", loginID).First(&userModel)
	return
}

func (useruser *User) Delete() (rowsAffected int64) {
	result := database.DB.Delete(&useruser)
	return result.RowsAffected
}
