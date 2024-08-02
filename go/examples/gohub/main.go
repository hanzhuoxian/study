package main

import (
	"fmt"
	"gohub/app/cmd"
	"gohub/app/cmd/make"
	"gohub/bootstrap"
	btsConfig "gohub/config"
	"gohub/pkg/config"
	"gohub/pkg/console"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	// 加载config目录下的配置信息
	btsConfig.Initialize()
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "gohub",
		Short: "gohub",
		Long:  `Default will run "serve" command, you can use "-h" flag to see all subcommands`,
		PersistentPreRun: func(command *cobra.Command, args []string) {

			config.InitConfig(cmd.Env)

			// 初始化 Logger
			bootstrap.SetupLogger()

			// gin.SetMode(gin.ReleaseMode)
			// 初始化数据库
			bootstrap.SetupDB()
			// 初始化 Redis
			bootstrap.SetupRedis()
			//初始化缓存
			bootstrap.SetupCache()
		},
	}

	rootCmd.AddCommand(
		cmd.CmdServe,
		cmd.CmdKey,
		cmd.CmdPlay,
		cmd.CmdMigrate,
		cmd.CmdDBSeed,
		cmd.CmdCache,
		// 生成命令
		make.CmdMake,
	)

	cmd.RegisterDefaultCmd(rootCmd, cmd.CmdServe)

	cmd.RegisterGlobalFlags(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		console.Exit(fmt.Sprintf("Failed to run app with %v: %s", os.Args, err.Error()))
	}
}
