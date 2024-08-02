package response

import (
	"gohub/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// JSON 响应 200 和 json 数据
func JSON(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

// Success 响应200 成功
func Success(c *gin.Context) {
	JSON(c, gin.H{
		"success": true,
		"message": "ok",
	})
}

// Data 响应 200 和 data 数据
func Data(c *gin.Context, data interface{}) {
	JSON(c, gin.H{
		"success": true,
		"data":    data,
	})
}

// Created 响应 201 和 data数据
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    data,
	})
}

// CreatedJson 响应 201 和 json 数据
func CreatedJson(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}

// Abort404 响应404
func Abort404(c *gin.Context, msg ...string) {
	c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
		"message": defaultMessage("404", msg...),
	})
}

// Abort403 响应403
func Abort403(c *gin.Context, msg ...string) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"message": defaultMessage("403", msg...),
	})
}

// Abort500 响应400
func Abort500(c *gin.Context, msg ...string) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
		"message": defaultMessage("500", msg...),
	})
}

// BadRequest 响应 400
func BadRequest(c *gin.Context, err error, msg ...string) {
	logger.LogIf(err)
	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
		"message": defaultMessage("400", msg...),
		"error":   err.Error(),
	})
}

// Error 响应 404 或 402
// 处理请求出错时出现 err ，会返回 error 信息
func Error(c *gin.Context, err error, msg ...string) {
	logger.LogIf(err)

	if err == gorm.ErrRecordNotFound {
		Abort404(c)
		return
	}

	c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
		"message": defaultMessage("请求处理失败", msg...),
		"error":   err.Error(),
	})
}

// ValidationError
func ValidationError(c *gin.Context, errors map[string][]string) {
	c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
		"message": " 请求验证不通过",
		"errors":  errors,
	})
}

// Unauthorized 响应 401 登录失败、jwt解析失败时使用
func Unauthorized(c *gin.Context, msg ...string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": defaultMessage("401", msg...)})
}

func defaultMessage(defaultMessage string, msg ...string) (message string) {
	if len(msg) > 0 {
		message = msg[0]
	} else {
		message = defaultMessage
	}
	return message
}
