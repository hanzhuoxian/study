package requests

import (
	"gohub/app/requests/validators"

	"github.com/gin-gonic/gin"
	"github.com/thedevsaddam/govalidator"
)

// VerifyCodeRequest 验证码请求结构
type VerifyCodeRequest struct {
	Phone         string `json:"phone,omitempty" valid:"phone"`
	CaptchaID     string `json:"captcha_id,omitempty" valid:"captcha_id"`
	CaptchaAnswer string `json:"captcha_answer,omitempty" valid:"captcha_answer"`
}

// VerifyCodePhone 验证手机号
func VerifyCodePhone(data interface{}, c *gin.Context) map[string][]string {
	// 1. 定制规则
	rules := govalidator.MapData{
		"phone":          []string{"required", "digits:11"},
		"captcha_id":     []string{"required"},
		"captcha_answer": []string{"required", "digits:6"},
	}

	// 定制错误消息
	messages := govalidator.MapData{
		"phone": []string{
			"required:手机号必填",
			"digits:手机号必须为 11 位数字",
		},
		"captcha_id": []string{
			"required:图片验证码 ID 必填",
		},
		"captcha_answer": []string{
			"required:验证码必填",
			"digits:验证码必须为6位数字",
		},
	}
	errs := validate(data, rules, messages)
	verifyCode := data.(*VerifyCodeRequest)

	errs = validators.ValidateCaptcha(verifyCode.CaptchaID, verifyCode.CaptchaAnswer, errs)
	return errs
}

type EmailVerifyCodeRequest struct {
	Email         string `json:"email" valid:"email"`
	CaptchaID     string `json:"captcha_id,omitempty" valid:"captcha_id"`
	CaptchaAnswer string `json:"captcha_answer,omitempty" valid:"captcha_answer"`
}

func VerifyCodeEmail(data interface{}, c *gin.Context) map[string][]string {
	// 1. 定制规则
	rules := govalidator.MapData{
		"email":          []string{"required", "min:4", "max:30", "email"},
		"captcha_id":     []string{"required"},
		"captcha_answer": []string{"required", "digits:6"},
	}

	// 定制错误消息
	messages := govalidator.MapData{
		"email": []string{
			"required:Email必填",
			"email: 请填入正确格式的邮件地址",
			"min: Email 长度需要大于 4 位",
			"max: Email 长度需要小于 30 位",
		},
		"captcha_id": []string{
			"required:图片验证码 ID 必填",
		},
		"captcha_answer": []string{
			"required:验证码必填",
			"digits:验证码必须为6位数字",
		},
	}
	errs := validate(data, rules, messages)
	// verifyCodeEmail := data.(*EmailVerifyCodeRequest)

	// errs = validators.ValidateCaptcha(verifyCodeEmail.CaptchaID, verifyCodeEmail.CaptchaAnswer, errs)

	return errs
}
