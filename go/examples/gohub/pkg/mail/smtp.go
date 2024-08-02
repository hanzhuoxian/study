package mail

import (
	"fmt"
	"gohub/pkg/logger"
	"net/smtp"

	"github.com/jordan-wright/email"
)

// SMTP SMTP 实例
type SMTP struct {
}

func (s *SMTP) Send(e Email, config map[string]string) bool {
	ne := email.NewEmail()
	ne.From = fmt.Sprintf("%v <%v>", e.From.Name, e.From.Adress)
	ne.To = e.To
	ne.Bcc = e.Bcc
	ne.Cc = e.Cc
	ne.Subject = e.Subject
	ne.Text = e.Text
	ne.HTML = e.HTML

	logger.DebugJSON("发送邮件", "发件详情", e)

	err := ne.Send(
		fmt.Sprintf("%v:%v", config["host"], config["port"]),
		smtp.PlainAuth(
			"",
			config["username"],
			config["password"],
			config["host"],
		),
	)

	if err != nil {
		logger.ErrorString("发送邮件", "发件出错", err.Error())
		return false
	}

	logger.DebugString("发送邮件", "发件成功", "")
	return true
}
