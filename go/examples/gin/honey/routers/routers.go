package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func HelloHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Hello q1mi!",
	})
}

func SetupRouter() *gin.Engine {
	r := gin.Default()
	r.GET("/hello", HelloHandler)
	return r
}
