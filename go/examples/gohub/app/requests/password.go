package requests

import (
	"gohub/app/requests/validators"

	"github.com/gin-gonic/gin"
	"github.com/thedevsaddam/govalidator"
)

// PasswordResetByPhone 手机重置密码请求结构体
type PasswordResetByPhone struct {
	Phone           string `json:"phone,omitempty" valid:"phone"`
	VerifyCode      string `json:"verify_code,omitempty" valid:"verify_code"`
	Password        string `json:"password,omitempty" valid:"password"`
	PasswordConfirm string `json:"password_confirm" valid:"password_confirm"`
}

// PasswordResetByEmail 邮箱重置密码请求结构体
type PasswordResetByEmail struct {
	Email           string `json:"email,omitempty" valid:"email"`
	VerifyCode      string `json:"verify_code,omitempty" valid:"verify_code"`
	Password        string `json:"password,omitempty" valid:"password"`
	PasswordConfirm string `json:"password_confirm" valid:"password_confirm"`
}

// PasswordResetByPhoneHandler 通过手机重置密码校验器
func PasswordResetByPhoneHandler(data interface{}, c *gin.Context) map[string][]string {
	rules := govalidator.MapData{
		"phone":            []string{"required", "digits:11"},
		"password":         []string{"required", "min:6"},
		"password_confirm": []string{"required"},
		"verify_code":      []string{"required", "digits:6"},
	}

	messages := govalidator.MapData{
		"phone": []string{
			"required:手机号为必填项，参数名称 phone",
			"digits:手机号长度必须为 11 位的数字",
		},
		"password": []string{
			"required:密码为必填项",
			"min:密码长度需大于 6",
		},
		"password_confirm": []string{
			"required:确认密码框为必填项",
		},
		"verify_code": []string{
			"required:验证码答案必填",
			"digits:验证码长度必须为 6 位的数字",
		},
	}
	errs := validate(data, rules, messages)
	password := data.(*PasswordResetByPhone)
	validators.ValidatePasswordConfirm(password.Password, password.PasswordConfirm, errs)
	errs = validators.ValidateVerifyCode(password.Phone, password.VerifyCode, errs)
	return errs
}

// PasswordResetByEmailHandler 通过邮箱重置密码校验器
func PasswordResetByEmailHandler(data interface{}, c *gin.Context) map[string][]string {
	rules := govalidator.MapData{
		"email":            []string{"required", "min:4", "max:30", "email"},
		"password":         []string{"required", "min:6"},
		"password_confirm": []string{"required"},
		"verify_code":      []string{"required", "digits:6"},
	}

	messages := govalidator.MapData{
		"email": []string{
			"required: Email 为必填项",
			"min: Email 长度需大于 4",
			"max: Email 长度需小于 30",
			"email: Email 格式不正确，请提供有效的邮箱地址",
		},
		"password": []string{
			"required:密码为必填项",
			"min:密码长度需大于 6",
		},
		"password_confirm": []string{
			"required:确认密码框为必填项",
		},
		"verify_code": []string{
			"required:验证码答案必填",
			"digits:验证码长度必须为 6 位的数字",
		},
	}
	errs := validate(data, rules, messages)
	password := data.(*PasswordResetByEmail)
	validators.ValidatePasswordConfirm(password.Password, password.PasswordConfirm, errs)
	errs = validators.ValidateVerifyCode(password.Email, password.VerifyCode, errs)
	return errs
}
