package sms

import (
	"encoding/json"
	"gohub/pkg/logger"

	aliyunsmsclient "github.com/KenmyZhang/aliyun-communicate"
)

type Aliyun struct{}

func (s *Aliyun) Send(phone string, message Message, config map[string]string) bool {
	smsClient := aliyunsmsclient.New("http://dysmsapi.aliyuncs.com")
	templateParam, err := json.Marshal(message.Data)
	if err != nil {
		logger.ErrorString("短信[阿里云]", "解析绑定错误", err.Error())
	}

	logger.DebugJSON("短信[阿里云]", "配置信息", config)
	logger.DebugJSON("短信[阿里云]", "请求内容", smsClient.Request)
	result, err := smsClient.Execute(
		config["access_key_id"],
		config["access_key_secret"],
		phone,
		config["sign_name"],
		message.Template,
		string(templateParam),
	)
	if err != nil {
		logger.ErrorString("短信[阿里云]", "发送短信失败", err.Error())
	}
	logger.DebugJSON("短信[阿里云]", "响应内容", result)

	resultJSON, err := json.Marshal(result)
	if err != nil {
		logger.ErrorString("短信[阿里云]", "解析响应 JSON 错误", err.Error())
	}

	if result.IsSuccessful() {
		logger.DebugString("短信[阿里云]", "发送短信成功", "")
		return true
	}
	logger.ErrorString("短信[阿里云]", "服务商返回错误", string(resultJSON))
	return false
}
