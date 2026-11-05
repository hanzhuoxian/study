package main

import (
	"log"
	"net/http"

	"github.com/casbin/casbin/v3"
	"github.com/gin-gonic/gin"
)

// Authorize asks Casbin whether the current user may call this route.
func Authorize(e *casbin.Enforcer) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Replace this with the user from your session or JWT.
		user := c.GetHeader("X-User")

		allowed, err := e.Enforce(user, c.Request.URL.Path, c.Request.Method)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}

func main() {
	e, err := casbin.NewEnforcer("model.conf", "policy.csv")
	if err != nil {
		log.Fatal(err)
	}

	r := gin.Default()
	api := r.Group("/api", Authorize(e))
	api.GET("/articles", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"articles": []string{}}) })
	api.POST("/articles", func(c *gin.Context) { c.JSON(http.StatusCreated, gin.H{"created": true}) })
	api.PUT("/articles/:id", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"updated": c.Param("id")}) })
	api.DELETE("/articles/:id", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"deleted": c.Param("id")}) })

	log.Fatal(r.Run(":8080"))
}
