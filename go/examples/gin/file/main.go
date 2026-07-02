package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	router.POST("/upload", func(ctx *gin.Context) {
		form, err := ctx.MultipartForm()
		if err != nil {
			ctx.JSON(http.StatusOK, gin.H{"error": err.Error()})
		}
		files := form.File["files"]
		for _, file := range files {
			log.Println(file.Filename)
		}
	})

	router.Run()
}
