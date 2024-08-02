// Package factories 存放工厂方法
package factories

import (
	"gohub/app/models/user"
	"gohub/helpers"

	"github.com/bxcodec/faker/v3"
)

func MakeUser(times int) []user.User {
	var objs []user.User

	faker.SetGenerateUniqueValues(true)

	for i := 0; i < times; i++ {
		model := user.User{
			Name:     faker.Username(),
			Email:    faker.Email(),
			Phone:    helpers.RandomNumber(11),
			Password: "$2a$14$Hh.ZKhX2jVv7RLJRuXklk.yWR1QBf.SrHg1JlSosI9dWP/XexpdS.",
		}
		objs = append(objs, model)
	}

	return objs
}
