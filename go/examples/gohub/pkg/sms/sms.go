package sms

import (
	"gohub/pkg/config"
	"sync"
)

// Message 短信结构体
type Message struct {
	Template string
	Data     map[string]string
	Content  string
}

// SMS 发送短信操作类
type SMS struct {
	Driver Driver
}

var once sync.Once

var internalSMS *SMS

func New() *SMS {
	once.Do(func() {
		internalSMS = &SMS{
			Driver: &Aliyun{},
		}
	})
	return internalSMS
}

func (sms *SMS) Send(phone string, message Message) bool {
	return sms.Driver.Send(phone, message, config.GetStringMapString("sms.aliyun"))
}
