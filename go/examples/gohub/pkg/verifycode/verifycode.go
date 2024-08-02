// Package verifycode 验证码
package verifycode

import (
	"fmt"
	"gohub/helpers"
	"gohub/pkg/app"
	"gohub/pkg/config"
	"gohub/pkg/logger"
	"gohub/pkg/mail"
	"gohub/pkg/redis"
	"gohub/pkg/sms"
	"strings"
	"sync"
)

// VerifyCode
type VerifyCode struct {
	Store Store
}

var once sync.Once
var internalVerifyCode *VerifyCode

// New 创建验证码实例
func New() *VerifyCode {
	once.Do(func() {
		internalVerifyCode = &VerifyCode{
			Store: &RedisStore{
				RedisClient: redis.Redis,
				KeyPrefix:   config.Get("app.name" + ":verifycode:"),
			},
		}
	})

	return internalVerifyCode
}

// SendSMS 发送短信
func (v *VerifyCode) SendSMS(phone string) bool {
	code := v.generateVerifyCode(phone)

	if !app.IsProduction() && strings.HasPrefix(phone, config.GetString("verfifycode.debug_phone_prefix")) {
		return true
	}
	return sms.New().Send(phone, sms.Message{
		Template: config.GetString("sms.aliyun.template_code"),
		Data:     map[string]string{"code": code},
	})
}

// Check 校验验证码
func (v *VerifyCode) Check(key, answer string) bool {
	return v.Store.Verify(key, answer, true)
}

func (v *VerifyCode) generateVerifyCode(key string) string {
	// 生成随机码
	code := helpers.RandomNumber(config.GetInt("verifycode.code_length"))
	// 开发环境处理
	if app.IsLocal() {
		code = config.GetString("verifycode.debug_code")
	}
	logger.DebugJSON("验证码", "生成验证码", map[string]string{key: code})
	v.Store.Set(key, code)
	return code
}

// SendMail 发送邮件验证码
func (v *VerifyCode) SendMail(email string) error {
	code := v.generateVerifyCode(email)
	contnet := fmt.Sprintf("<h1>你的邮件验证码是%v</h>", code)
	mail.New().Send(mail.Email{
		From: mail.From{
			Adress: config.GetString("mail.from.address"),
			Name:   config.GetString("mail.from.name"),
		},
		To:      []string{email},
		Subject: "Email 验证码",
		HTML:    []byte(contnet),
	})
	return nil
}
