package main

import (
	"log"
	"time"

	router "github.com/daymenu/gostudy/examples/gin/honey/routers"
	ginzap "github.com/gin-contrib/zap"
	"go.uber.org/zap"
)

func main() {
	app := router.SetupRouter()
	logger, _ := zap.NewProduction()

	// Add a ginzap middleware, which:
	//   - Logs all requests, like a combined access and error log.
	//   - Logs to stdout.
	//   - RFC3339 with UTC time format.
	app.Use(ginzap.Ginzap(logger, time.RFC3339, true))

	// Logs all panic to error log
	//   - stack means whether output the stack info.
	app.Use(ginzap.RecoveryWithZap(logger, true))
	if err := app.Run(); err != nil {
		log.Fatal(app)
	}
}
