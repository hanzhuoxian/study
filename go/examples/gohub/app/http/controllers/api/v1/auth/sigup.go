// Package auth 处理用户身份认证相关逻辑
package auth

import (
	v1 "gohub/app/http/controllers/api/v1"
	"gohub/app/models/user"
	"gohub/app/requests"
	"gohub/pkg/jwt"
	"gohub/pkg/response"

	"github.com/gin-gonic/gin"
)

// SignupController
type SignupController struct {
	v1.BaseAPIController
}

// IsPhoneExist 检测手机号是否被注册
func (s *SignupController) IsPhoneExist(c *gin.Context) {
	request := requests.SignupPhoneExistRequest{}

	if ok, errs := requests.Validate(c, &request, requests.SignupPhoneExist); !ok {

		response.ValidationError(c, errs)
		return
	}

	// 检查数据库并返回响应
	response.JSON(c, gin.H{
		"exist": user.IsPhoneExist(request.Phone),
	})
}

func (s *SignupController) IsEmailExist(c *gin.Context) {
	request := requests.SignupEmailExistRequest{}

	if ok, errs := requests.Validate(c, &request, requests.SignupEmailExist); !ok {
		response.ValidationError(c, errs)
		return
	}

	// 检查数据库并返回响应
	response.JSON(c, gin.H{
		"exist": user.IsEmailExist(request.Email),
	})
}

// SignupUsingPhone 使用手机号注册
func (s *SignupController) SignupUsingPhone(c *gin.Context) {
	r := requests.SignupUsingPhoneRequest{}
	if ok, errs := requests.Validate(c, &r, requests.SignupUsingPhone); !ok {
		response.ValidationError(c, errs)
		return
	}

	userModel := user.User{
		Name:     r.Name,
		Phone:    r.Phone,
		Password: r.Password,
	}

	userModel.Create()

	if userModel.ID > 0 {
		token := jwt.New().IssueToken(userModel.GetStringID(), userModel.Name)
		response.CreatedJson(c, gin.H{
			"data":  userModel,
			"token": token,
		})
		return
	}

	response.Abort500(c, "创建用户失败，请稍候重试")
}

// SignupUsingEmail 使用邮箱注册
func (s *SignupController) SignupUsingEmail(c *gin.Context) {
	r := requests.SignupUsingEmailRequest{}
	if ok, errs := requests.Validate(c, &r, requests.SignupUsingEmail); !ok {
		response.ValidationError(c, errs)
		return
	}

	userModel := user.User{
		Name:     r.Name,
		Email:    r.Email,
		Password: r.Password,
	}

	userModel.Create()

	if userModel.ID > 0 {
		token := jwt.New().IssueToken(userModel.GetStringID(), userModel.Name)
		response.CreatedJson(c, gin.H{
			"data":  userModel,
			"token": token,
		})
		return
	}

	response.Abort500(c, "创建用户失败，请稍候重试")

}
