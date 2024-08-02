package model

type User struct {
	Name     string `form:"name"`
	Password string `form:"password"`
	Orders   []Order
}

type Order struct {
	Product string `form:"product"`
	UserID  uint
}
