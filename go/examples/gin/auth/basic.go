package main

import (
	"encoding/base64"
	"strings"

	"github.com/gin-gonic/gin"
)

type BasicStrategy struct {
	compare func(username string, password string) bool
}

var _ AuthStrategy = &BasicStrategy{}

func NewBasicStrategy(compare func(username string, password string) bool) AuthStrategy {
	return &BasicStrategy{compare: compare}
}

func (b *BasicStrategy) AuthFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := strings.SplitN(c.Request.Header.Get("Authorization"), " ", 2)
		if len(auth) != 2 || auth[0] != "Basic" {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid authorization header"})
			return
		}
		payload, _ := base64.StdEncoding.DecodeString(auth[1])
		creds := strings.SplitN(string(payload), ":", 2)
		if len(creds) != 2 {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid credentials"})
			return
		}
		if !b.compare(creds[0], creds[1]) {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid credentials"})
			return
		}
		c.Set(CtxUserID, creds[0])
		c.Next()
	}
}

func NewBasicAuth(compare func(username string, password string) bool) AuthStrategy {
	return &BasicStrategy{compare: compare}
}
