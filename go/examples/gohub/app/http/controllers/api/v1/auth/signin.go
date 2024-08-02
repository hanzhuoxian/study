package auth

import (
	v1 "gohub/app/http/controllers/api/v1"
	"gohub/app/requests"
	"gohub/pkg/auth"
	"gohub/pkg/jwt"
	"gohub/pkg/response"

	"github.com/gin-gonic/gin"
)

// SigninController
type SigninController struct {
	v1.BaseAPIController
}

// LoginByPhone 手机号登录
func (s *SigninController) LoginByPhone(c *gin.Context) {
	r := requests.LoginByPhoneRequest{}
	if ok, errs := requests.Validate(c, &r, requests.LoginByPhone); !ok {
		response.ValidationError(c, errs)
		return
	}

	user, err := auth.LoginByPhone(r.Phone)
	if err != nil {
		response.Error(c, err, "账号不存在")
		return
	}

	token := jwt.New().IssueToken(user.GetStringID(), user.Name)
	response.JSON(c, gin.H{
		"token": token,
	})
}

// Login 登录
func (s *SigninController) Login(c *gin.Context) {
	r := requests.LoginRequest{}
	if ok, errs := requests.Validate(c, &r, requests.Login); !ok {
		response.ValidationError(c, errs)
		return
	}
	user, err := auth.Attempt(r.LoginID, r.Password)
	if err != nil {
		response.Error(c, err, "账号不存在或密码错误")
	}
	token := jwt.New().IssueToken(user.GetStringID(), user.Name)
	response.JSON(c, gin.H{
		"token": token,
	})
}

// RefreshToken token 刷新
func (s *SigninController) RefreshToken(c *gin.Context) {
	if token, err := jwt.New().RefreshToken(c); err != nil {
		response.Error(c, err, "令牌刷新失败")
	} else {
		response.JSON(c, gin.H{
			"token": token,
		})
	}

}
