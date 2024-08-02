package main

import (
	"fmt"
	"net/http"

	"github.com/daymenu/gostudy/examples/learnku/ginbind/model"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// HTTP 请求方法修改
	r.POST("/api/v1/get", func(ctx *gin.Context) {
		var query model.User

		ctx.ShouldBind(&query)

		fmt.Println(query)

		// result := db.Get(&query)

		ctx.JSON(http.StatusOK, gin.H{
			"data": query,
		})
	})

	r.Run(":80")
}
