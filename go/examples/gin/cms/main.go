package main

import (
	"github.com/gin-gonic/gin"
	"github.com/hanzhuoxian/study/go/examples/gin/cms/bootstrap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type User struct {
	Name string `json:"name" binding:"required"`
	Age  int    `json:"age" binding:"required,min=0"`
}

func main() {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)     // 支持运行时动态调整级别
	config.OutputPaths = []string{"stdout", "logs/app.log"} // 同时输出到控制台和文件
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder // 人类易读的时间格式

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	router := gin.New()
	bootstrap.SetupRoute(router)
	router.Run() // listens on 0.0.0.0:8080 by default
}
