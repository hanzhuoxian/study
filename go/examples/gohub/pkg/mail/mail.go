package mail

import (
	"gohub/pkg/config"
	"sync"
)

// From 发件人信息
type From struct {
	Adress string
	Name   string
}

// Email Email
type Email struct {
	From    From
	To      []string
	Bcc     []string
	Cc      []string
	Subject string
	Text    []byte // Plaintext message (optional)
	HTML    []byte // HTML messagea (optional)
}

// Mailer Mailer
type Mailer struct {
	Driver Driver
}

var once sync.Once

var internalMailer *Mailer

// New 创建邮件实例
func New() *Mailer {
	once.Do(func() {
		internalMailer = &Mailer{
			Driver: &SMTP{},
		}
	})
	return internalMailer
}

func (m *Mailer) Send(email Email) bool {
	return m.Driver.Send(email, config.GetStringMapString("mail.smtp"))
}
