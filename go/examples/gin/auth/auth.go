package main

import (
	"github.com/gin-gonic/gin"
)

const (
	ServerAudience = "gin.auth"
	ServerIssuer   = "auth"
	ServerKey      = "88888888"
	CtxUserID      = "user_id"
)

type User struct {
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthStrategy interface {
	AuthFunc() gin.HandlerFunc
}

type AuthOperator struct {
	strategy AuthStrategy
}

func (a *AuthOperator) SetStrategy(strategy AuthStrategy) {
	a.strategy = strategy
}

func (a *AuthOperator) AuthFunc() gin.HandlerFunc {
	return a.strategy.AuthFunc()
}
