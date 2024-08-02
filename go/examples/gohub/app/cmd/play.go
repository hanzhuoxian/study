package cmd

import (
	"gohub/pkg/console"
	"gohub/pkg/redis"
	"time"

	"github.com/spf13/cobra"
)

var CmdPlay = &cobra.Command{
	Use:   "play",
	Short: "Likes the go playground, but runging at our application context",
	Run:   runPlay,
}

func runPlay(cmd *cobra.Command, args []string) {
	redis.Redis.Set("hello", "hi", 10*time.Minute)
	console.Success(redis.Redis.Get("hello"))
}
