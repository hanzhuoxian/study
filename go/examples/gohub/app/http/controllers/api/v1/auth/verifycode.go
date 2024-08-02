package auth

import (
	v1 "gohub/app/http/controllers/api/v1"
	"gohub/app/requests"
	"gohub/pkg/captcha"
	"gohub/pkg/logger"
	"gohub/pkg/response"
	"gohub/pkg/verifycode"

	"github.com/gin-gonic/gin"
)

// VerifyCodeController 验证码控制器
type VerifyCodeController struct {
	v1.BaseAPIController
}

// ShowCaptcha 展示图片二维码
func (v *VerifyCodeController) ShowCaptcha(c *gin.Context) {
	id, b64s, err := captcha.New().GenerateCaptcha()
	logger.LogIf(err)
	response.JSON(c, gin.H{
		"captcha_id":    id,
		"captcha_image": b64s,
	})
}

// SendUsingPhone 发送短信验证码
func (v *VerifyCodeController) SendUsingPhone(c *gin.Context) {
	// 验证表单
	request := requests.VerifyCodeRequest{}
	if ok, errs := requests.Validate(c, &request, requests.VerifyCodePhone); !ok {
		response.ValidationError(c, errs)
		return
	}
	// 发送短信
	if ok := verifycode.New().SendSMS(request.Phone); !ok {
		response.Abort500(c, "发送短信失败")
		return
	}
	response.Success(c)
}

// SendMail 发送邮件验证码
func (v *VerifyCodeController) SendMail(c *gin.Context) {
	r := requests.EmailVerifyCodeRequest{}
	if ok, errs := requests.Validate(c, &r, requests.VerifyCodeEmail); !ok {
		response.ValidationError(c, errs)
		return
	}
	if err := verifycode.New().SendMail(r.Email); err != nil {
		response.Abort500(c, "发送邮件错误")
	}
	response.Success(c)
}
