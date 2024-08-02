package app

import (
	"gohub/pkg/config"
	"time"
)

// IsLocal 是否是本地开发环境
func IsLocal() bool {
	return config.Get("app.env") == "local"
}

// IsProduction 是否是生产开发环境
func IsProduction() bool {
	return config.Get("app.env") == "production"
}

// IsTesting 是否是测试生产环境
func IsTesting() bool {
	return config.Get("app.env") == "testing"
}

// TimenowInTimezone 获取匹配时区的时间
func TimenowInTimezone() time.Time {
	chinaTimezone, _ := time.LoadLocation(config.GetString("app.timezone"))
	return time.Now().In(chinaTimezone)
}

// URL 传参 path 拼接站点的 URL
func URL(path string) string {
	return config.Get("app.url") + path
}

// V1URL 拼接带 v1 标示 URL
func V1URL(path string) string {
	return URL("/v1/" + path)
}
