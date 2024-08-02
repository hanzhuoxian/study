// Package middlewares 存放系统中间件
package middlewares

import (
	"bytes"
	"gohub/helpers"
	"gohub/pkg/logger"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"go.uber.org/zap"
)

type responseBodyWrite struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r responseBodyWrite) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// Logger 记录请求日志
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {

		// 获取 response 内容
		w := &responseBodyWrite{body: &bytes.Buffer{}, ResponseWriter: c.Writer}
		c.Writer = w

		// 获取请求数据
		var requestBody []byte
		if c.Request.Body != nil {
			// c.Request.Body 是一个 Buffer 对象，只能读取一次
			requestBody, _ = io.ReadAll(c.Request.Body)
			// 读取后重新赋值
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}
		// 设置开始时间
		start := time.Now()

		c.Next()

		// 开始记录日志的逻辑
		cost := time.Since(start)
		responseStatus := c.Writer.Status()

		logFields := []zap.Field{
			zap.Int("status", responseStatus),
			zap.String("request", c.Request.Method+c.Request.URL.RequestURI()),
			zap.String("query", c.Request.URL.RawQuery),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()),
			zap.String("time", helpers.MicrosecondStr(cost)),
		}

		switch c.Request.Method {
		case http.MethodPost, http.MethodPut, http.MethodDelete:
			// 添加请求内容
			logFields = append(logFields, zap.String("Request Body", string(requestBody)))
			// 响应的内容
			logFields = append(logFields, zap.String("Response Body", w.body.String()))
		}

		if responseStatus > 400 && responseStatus <= 499 {
			logger.Warn("HTTP Warning"+cast.ToString(responseStatus), logFields...)
		} else if responseStatus > 500 && responseStatus <= 599 {
			logger.Error("HTTP Error"+cast.ToString(responseStatus), logFields...)
		} else {
			logger.Debug("HTTP access log", logFields...)
		}
	}
}
