// Package mail 邮件
package mail

type Driver interface {
	Send(email Email, config map[string]string) bool
}
