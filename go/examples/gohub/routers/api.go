// Package routers 注册路由

package routers

import (
	controllers "gohub/app/http/controllers/api/v1"
	"gohub/app/http/controllers/api/v1/auth"
	"gohub/app/http/middlewares"
	"gohub/pkg/config"

	"github.com/gin-gonic/gin"
)

// RegisterAPIRouters 注册相关路由
func RegisterAPIRouters(r *gin.Engine) {
	// 测试一个 v1 的路由组，我们所有的 v1 版本的路由都将存放到这里
	var v1 *gin.RouterGroup
	if len(config.Get("app.api_domain")) == 0 {
		v1 = r.Group("/api/v1")
	} else {
		v1 = r.Group("/v1")
	}

	v1.Use(middlewares.LimitIP("200-H"))
	{
		authGroup := v1.Group("/auth")
		{
			// 注册
			signup := new(auth.SignupController)
			authGroup.POST("/signup/phone/exist", signup.IsPhoneExist)
			authGroup.POST("/signup/email/exist", signup.IsEmailExist)
			authGroup.POST("/signup/using/phone", signup.SignupUsingPhone)
			authGroup.POST("/signup/using/email", signup.SignupUsingEmail)

			// 验证码
			vcc := new(auth.VerifyCodeController)
			authGroup.POST("/verify-codes/captcha", middlewares.LimitPerRoute("50-H"), vcc.ShowCaptcha)
			authGroup.POST("/verify-codes/phone", middlewares.LimitPerRoute("20-H"), vcc.SendUsingPhone)
			authGroup.POST("/verify-codes/email", middlewares.LimitPerRoute("20-H"), vcc.SendMail)

			// 登录
			signin := new(auth.SigninController)
			authGroup.POST("/login/phone", signin.LoginByPhone)
			authGroup.POST("/login/user", signin.Login)
			authGroup.POST("/login/refresh", signin.RefreshToken)

			// 重置密码
			password := new(auth.PasswordController)
			authGroup.POST("/password-reset/phone", password.ResetByPhone)
			authGroup.POST("/password-reset/email", password.ResetByEmail)

			// 获取当前用户
			uc := new(controllers.UsersController)
			v1.GET("/user", middlewares.AuthJWT(), uc.CurrentUser)
			usersGroup := v1.Group("/users")
			{
				usersGroup.GET("", uc.Index)
				usersGroup.PUT("", middlewares.AuthJWT(), uc.UpdateProfile)
				usersGroup.PUT("/email", middlewares.AuthJWT(), uc.UpdateEmail)
				usersGroup.PUT("/phone", middlewares.AuthJWT(), uc.UpdatePhone)
				usersGroup.PUT("/password", middlewares.AuthJWT(), uc.UpdatePassword)
				usersGroup.PUT("/avatar", middlewares.AuthJWT(), uc.UpdateAvatar)
			}
			// 类别
			cgc := new(controllers.CategoriesController)
			cgcGroup := v1.Group("/categories")
			{
				cgcGroup.POST("", middlewares.AuthJWT(), cgc.Store)
				cgcGroup.PUT("/:id", middlewares.AuthJWT(), cgc.Update)
				cgcGroup.GET("", cgc.Index)
				cgcGroup.DELETE("/:id", middlewares.AuthJWT(), cgc.Delete)
			}

			tpc := new(controllers.TopicsController)
			tpcGroup := v1.Group("/topics")
			{
				tpcGroup.POST("", middlewares.AuthJWT(), tpc.Store)
				tpcGroup.PUT("/:id", middlewares.AuthJWT(), cgc.Update)
				tpcGroup.GET("", cgc.Index)
				tpcGroup.DELETE("/:id", middlewares.AuthJWT(), cgc.Delete)
			}

			lsc := new(controllers.LinksController)
			linksGroup := v1.Group("/links")
			{
				linksGroup.GET("", lsc.Index)
			}
		}

	}
}
