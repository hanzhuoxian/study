// Package limiter 限流
package limiter

import (
	"gohub/pkg/config"
	"gohub/pkg/logger"
	"gohub/pkg/redis"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	limitredis "github.com/ulule/limiter/v3/drivers/store/redis"
)

// GetKeyIP 获取客户端IP
func GetKeyIP(c *gin.Context) string {
	return c.ClientIP()
}

// GetRouterWithIP 接口+IP 限流
func GetRouterWithIP(c *gin.Context) string {
	return routerToKeyString(c.FullPath() + GetKeyIP(c))
}

func CheckRate(c *gin.Context, key string, formatted string) (limiter.Context, error) {
	var context limiter.Context
	rate, err := limiter.NewRateFromFormatted(formatted)
	if err != nil {
		logger.LogIf(err)
		return context, err
	}
	store, err := limitredis.NewStoreWithOptions(redis.Redis.Client, limiter.StoreOptions{
		Prefix: config.GetString("app.name") + ":limiter:",
	})

	limit := limiter.New(store, rate)

	if c.GetBool("limiter-once") {
		return limit.Peek(c, key)
	} else {
		c.Set("limiter-once", true)
		return limit.Get(c, key)
	}
}

func routerToKeyString(routeName string) string {
	routeName = strings.ReplaceAll(routeName, "/", "-")
	routeName = strings.ReplaceAll(routeName, ":", "_")
	return routeName
}
