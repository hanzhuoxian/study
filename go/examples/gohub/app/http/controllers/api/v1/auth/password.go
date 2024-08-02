package auth

import (
	v1 "gohub/app/http/controllers/api/v1"
	"gohub/app/models/user"
	"gohub/app/requests"
	"gohub/pkg/response"

	"github.com/gin-gonic/gin"
)

// PasswordController PasswordController
type PasswordController struct {
	v1.BaseAPIController
}

// ResetByPhone ResetByPhone
func (p *PasswordController) ResetByPhone(c *gin.Context) {
	r := requests.PasswordResetByPhone{}
	if ok, errs := requests.Validate(c, &r, requests.PasswordResetByPhoneHandler); !ok {
		response.ValidationError(c, errs)
		return
	}

	userModel := user.GetByPhone(r.Phone)
	if userModel.ID == 0 {
		response.Abort404(c)
	}

	userModel.Password = r.Password
	userModel.Save()
	response.Success(c)
}

// ResetByEmail 通过邮箱重置密码控制器
func (p *PasswordController) ResetByEmail(c *gin.Context) {
	r := requests.PasswordResetByEmail{}
	if ok, errs := requests.Validate(c, &r, requests.PasswordResetByEmailHandler); !ok {

		response.ValidationError(c, errs)
		return
	}

	userModel := user.GetByEmail(r.Email)
	if userModel.ID == 0 {
		response.Abort404(c)
		return
	}

	userModel.Password = r.Password
	userModel.Save()
	response.Success(c)
}
