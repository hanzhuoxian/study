package cmd

import (
	"fmt"
	"gohub/pkg/cache"
	"gohub/pkg/console"

	"github.com/spf13/cobra"
)

func init() {
	CmdCache.AddCommand(CmdCacheClear)
	CmdCache.AddCommand(CmdCacheForget)
	CmdCacheForget.Flags().StringVarP(&cacheKey, "key", "k", "", "KEY of the cache")
	CmdCacheForget.MarkFlagRequired("key")
}

var cacheKey string

var CmdCache = &cobra.Command{
	Use:   "cache",
	Short: "缓存相关命令",
}

var CmdCacheClear = &cobra.Command{
	Use:   "clear",
	Short: "清空缓存",
	Run:   runCacheClear,
}

func runCacheClear(cmd *cobra.Command, args []string) {
	cache.Flush()
	console.Success("cache cleared")
}

var CmdCacheForget = &cobra.Command{
	Use:   "forget",
	Short: "清空某个key",
	Run:   runCacheForget,
}

func runCacheForget(cmd *cobra.Command, args []string) {
	cache.Forget(cacheKey)
	console.Success(fmt.Sprintf("Cache key [%s] deleted.", cacheKey))
}
