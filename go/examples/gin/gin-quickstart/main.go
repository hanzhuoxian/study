package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

type Login struct {
	User     string `form:"user" json:"user" xml:"user"  binding:"required"`
	Password string `form:"password" json:"password" xml:"password" binding:"required"`
}

// Booking contains binded and validated data.
type Booking struct {
	CheckIn  time.Time `form:"check_in" json:"check_in" binding:"required,bookabledate" time_format:"2006-01-02"`
	CheckOut time.Time `form:"check_out" json:"check_out" binding:"required,gtfield=CheckIn,bookabledate" time_format:"2006-01-02"`
}

var bookableDate validator.Func = func(fl validator.FieldLevel) bool {
	date, ok := fl.Field().Interface().(time.Time)
	if ok {
		today := time.Now()
		if today.After(date) {
			return false
		}
	}
	return true
}

func main() {
	router := gin.Default()
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	// curl -X  GET http://localhost:8080/user
	router.GET("/user", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"get method": c.Request.Method})
	})
	// curl -X  POST http://localhost:8080/user
	router.POST("/user", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"post method": c.Request.Method})
	})
	// curl -X  PUT http://localhost:8080/user
	router.PUT("/user", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"put method": c.Request.Method})
	})
	// curl -X  PATCH http://localhost:8080/user
	router.PATCH("/user", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"patch method": c.Request.Method})
	})
	// curl -X  DELETE http://localhost:8080/user
	router.DELETE("/user", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"delete method": c.Request.Method})
	})
	// curl -X  OPTIONS http://localhost:8080/user
	router.OPTIONS("/user", func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "http://localhost:3000" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD")
			c.Header("Access-Control-Allow-Headers", "Content-Type")
			c.Header("Access-Control-Max-Age", "600")
		}
		c.Status(http.StatusNoContent)
	})
	// curl -v http://localhost:8080/user/edit
	router.HEAD("/user", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"head method": c.Request.Method})
	})

	// curl -v http://localhost:8080/user/john/edit
	router.GET("/user/:name/*action", func(c *gin.Context) {
		name := c.Param("name")
		action := c.Param("action")
		c.JSON(http.StatusOK, gin.H{"name": name, "action": action})
	})

	// curl -v http://localhost:8080/welcome?firstname=john&lastname=doe
	// curl -v http://localhost:8080/welcome?lastname=doe
	router.GET("/welcome", func(c *gin.Context) {
		firstname := c.DefaultQuery("firstname", "Guest")
		lastname := c.Query("lastname") // shortcut for c.Request.URL.Query().Get("lastname")

		c.String(http.StatusOK, "Hello %s %s", firstname, lastname)
	})
	// curl -X POST http://localhost:8080/form_post -d "message=hello&nick=world"
	// curl -X POST http://localhost:8080/form_post -F "message=hello" -F "nick=world" -F "foods=apple" -F "foods=banana" -F "person[a]=1" -F "person[b]=2"
	router.POST("/form_post", func(c *gin.Context) {
		message := c.PostForm("message")
		nick := c.DefaultPostForm("nick", "anonymous")
		foods := c.PostFormArray("foods")
		person := c.PostFormMap("person")

		c.JSON(200, gin.H{
			"status":  "posted",
			"message": message,
			"nick":    nick,
			"foods":   foods,
			"person":  person,
		})
	})

	// curl -X POST 'http://localhost:8080/post?ids%5Ba%5D=1234&ids%5Bb%5D=hello' -d 'names%5Bfirst%5D=thinkerou&names%5Bsecond%5D=tianou'
	router.POST("/post", func(c *gin.Context) {
		ids := c.QueryMap("ids")
		names := c.PostFormMap("names")

		fmt.Printf("ids: %v; names: %v\n", ids, names)
		c.JSON(http.StatusOK, gin.H{
			"ids":   ids,
			"names": names,
		})
	})

	router.MaxMultipartMemory = 8 << 20 // 8 MiB

	// curl -X POST 'http://localhost:8080/upload' -F 'file=@/etc/hosts'
	router.POST("/upload", func(c *gin.Context) {
		// single file
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.Println(file.Filename)
		if err := os.MkdirAll("./files", 0755); err != nil {
			return
		}
		// Upload the file to specific dst.
		dst := filepath.Join("./files/", filepath.Base(file.Filename))
		c.SaveUploadedFile(file, dst)

		c.String(http.StatusOK, fmt.Sprintf("'%s' uploaded!", file.Filename))
	})

	maxSize := int64(16 << 20) // 16MB
	// curl -X POST 'http://localhost:8080/upload/multi' -F 'files=@/etc/hosts' -F 'files=@/etc/passwd'
	router.POST("/upload/multi", func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)

		// Multipart form
		form, err := c.MultipartForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		files := form.File["files"]

		if os.Mkdir("./files/", 0755) != nil {
			return
		}
		for _, file := range files {
			log.Println(file.Filename)

			// Upload the file to specific dst.
			dst := filepath.Join("./files/", filepath.Base(file.Filename))
			c.SaveUploadedFile(file, dst)
		}
		c.String(http.StatusOK, fmt.Sprintf("%d files uploaded!", len(files)))
	})

	// curl -X GET http://localhost:8080/v1/login
	v1 := router.Group("/v1")
	{
		v1.GET("/login", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "login v1",
			})
		})
		user := v1.Group("/user")
		{
			// curl -X GET http://localhost:8080/v1/user/123
			user.GET("/:id", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"message": "get user v1",
				})
			})

			// curl -v -X POST http://localhost:8080/v1/user
			// curl -v -L -X POST http://localhost:8080/v1/user
			user.POST("/", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"message": "create user v1",
				})
			})
		}
	}
	// curl -X GET http://localhost:8080/v2/login
	v2 := router.Group("/v2")
	v2.GET("/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "login v2",
		})
	})
	// curl -X POST http://localhost:8080/loginJSON -H 'content-type: application/json' -d '{"user": "manu", "password": "123"}'
	// Example for binding JSON ({"user": "manu", "password": "123"})
	router.POST("/loginJSON", func(c *gin.Context) {
		var json Login
		if err := c.ShouldBind(&json); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if json.User != "manu" || json.Password != "123" {
			c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "you are logged in", "user": json.User})
	})

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("bookabledate", bookableDate)
	}

	// curl -X POST http://localhost:8080/bookable -H 'content-type: application/json' -d '{"check_in": "2026-10-29T00:00:00Z", "check_out": "2026-10-30T00:00:00Z"}'
	router.POST("/bookable", func(c *gin.Context) {
		var b Booking
		if err := c.ShouldBindJSON(&b); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"book": b})
	})

	router.Run() // listens on 0.0.0.0:8080 by default
}
