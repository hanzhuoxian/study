package user

import (
	"gohub/pkg/app"
	"gohub/pkg/database"
	"gohub/pkg/paginator"

	"github.com/gin-gonic/gin"
)

// IsEmailExist 判断 Email 是否被注册
func IsEmailExist(email string) bool {
	var count int64
	database.DB.Model(User{}).Where("email = ?", email).Count(&count)
	return count > 0
}

func IsPhoneExist(phone string) bool {
	var count int64
	database.DB.Model(User{}).Where("phone = ?", phone).Count(&count)
	return count > 0
}

func Get(idstr string) (user User) {
	database.DB.Where("id", idstr).First(&user)
	return
}

func GetBy(field, value string) (user User) {
	database.DB.Where("? = ?", field, value).First(&user)
	return
}

func All() (userS []User) {
	database.DB.Find(&userS)
	return
}

func IsExist(field, value string) bool {
	var count int64
	database.DB.Model(User{}).Where("? = ?", field, value).Count(&count)
	return count > 0
}

func Paginate(c *gin.Context, perPage int) (users []User, paging paginator.Paging) {
	paging = paginator.Paginate(c,
		database.DB.Model(User{}),
		&users,
		app.V1URL(database.TableName(&User{})),
		perPage,
	)
	return
}
