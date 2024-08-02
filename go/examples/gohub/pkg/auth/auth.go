// Package auth 授权相关逻辑
package auth

import (
	"errors"
	"gohub/app/models/user"
	"gohub/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Attempt 尝试登录 只验证是否可以登录
func Attempt(loginKey string, password string) (user.User, error) {
	userModel := user.GetByMulti(loginKey)
	if userModel.ID == 0 {
		return user.User{}, errors.New("账号不存在")
	}
	if !userModel.ComparePassword(password) {
		return user.User{}, errors.New("密码错误")
	}

	return userModel, nil
}

// LoginByPhone 通过手机号登录
func LoginByPhone(phone string) (user.User, error) {
	userModel := user.GetByPhone(phone)
	if userModel.ID == 0 {
		return user.User{}, errors.New("手机号未注册")
	}
	return userModel, nil
}

// CurrentUser 获取当前登录用户的模型
func CurrentUser(c *gin.Context) (userModel user.User) {
	userModel, ok := c.MustGet("current_user").(user.User)
	if !ok {
		logger.LogIf(errors.New("无法获取用户"))
		return user.User{}
	}
	return userModel
}

// CurrentUID 获取当前登录用户 ID
func CurrentUID(c *gin.Context) string {
	userID, ok := c.MustGet("current_user_id").(string)
	if !ok {
		logger.LogIf(errors.New("无法获取用户"))
		return ""
	}
	return userID
}
