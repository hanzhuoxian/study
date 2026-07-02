package main

import (
	"net/http"
	"time"

	jwt "github.com/appleboy/gin-jwt/v3"
	"github.com/gin-gonic/gin"
	gojwt "github.com/golang-jwt/jwt/v5"
)

func main() {
	r := gin.Default()
	jwtStrategy, _ := NewJWTAuth().(*JWTStrategy)
	jwtStrategy.MiddlewareInit()
	r.GET("/login", jwtStrategy.LoginHandler)
	r.GET("/logout", jwtStrategy.LogoutHandler)
	auth := r.Group("/auth")
	auth.Use(jwtStrategy.AuthFunc())
	auth.GET("userinfo", userinfo)
	r.Run()
}

func userinfo(c *gin.Context) {
	user, exists := c.Get(CtxUserID)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "no login",
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": user,
	})
}
func NewBasicAuthor() AuthStrategy {
	return NewBasicStrategy(func(username string, password string) bool {
		return username == "admin" && password == "admin"
	})
}

func NewJWTAuth() AuthStrategy {
	return NewJWTStrategy(initParams())
}

func initParams() jwt.GinJWTMiddleware {
	return jwt.GinJWTMiddleware{
		Realm:       "auth",
		Key:         []byte(ServerKey),
		Timeout:     time.Hour,
		MaxRefresh:  time.Hour,
		IdentityKey: CtxUserID,
		PayloadFunc: payloadFunc(),

		IdentityHandler:       identityHandler(),
		Authenticator:         authenticator(),
		Unauthorized:          unauthorized(),
		LogoutResponse:        logoutResponse(),
		HTTPStatusMessageFunc: httpStatusMessageFunc(),
		TokenLookup:           "header: Authorization, query: token, cookie: jwt",
		TokenHeadName:         "Bearer",
		SigningAlgorithm:      "HS256",
		TimeFunc:              time.Now,
	}
}

func httpStatusMessageFunc() func(c *gin.Context, err error) string {
	return func(c *gin.Context, err error) string {
		return err.Error()
	}
}

func payloadFunc() func(data any) gojwt.MapClaims {
	return func(data any) gojwt.MapClaims {
		if v, ok := data.(*User); ok {
			return gojwt.MapClaims{
				CtxUserID: v.Name,
			}
		}
		return gojwt.MapClaims{}
	}
}

func identityHandler() func(c *gin.Context) any {
	return func(c *gin.Context) any {
		claims := jwt.ExtractClaims(c)
		return &User{
			Name: claims[CtxUserID].(string),
		}
	}
}

func authenticator() func(c *gin.Context) (any, error) {
	return func(c *gin.Context) (any, error) {
		var loginVals User

		if err := c.BindJSON(&loginVals); err != nil {
			return nil, jwt.ErrMissingLoginValues
		}
		if loginVals.Name != "admin" || loginVals.Password != "admin" {
			return nil, jwt.ErrFailedAuthentication
		}
		return &User{Name: loginVals.Name}, nil
	}
}

func unauthorized() func(c *gin.Context, code int, message string) {
	return func(c *gin.Context, code int, message string) {
		c.JSON(code, gin.H{
			"code":    code,
			"message": message,
		})
	}
}

func logoutResponse() func(c *gin.Context) {
	return func(c *gin.Context) {
		claims := jwt.ExtractClaims(c)
		user, exists := c.Get(CtxUserID)
		response := gin.H{
			"code":    http.StatusOK,
			"message": "Successfully logged out",
		}
		if len(claims) > 0 {
			response["logged_out"] = claims[CtxUserID]
		}

		if exists {
			response["user_info"] = user.(*User).Name
		}

		c.JSON(http.StatusOK, response)
	}
}
