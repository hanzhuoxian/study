package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

// 声明一个全局的redisDb变量
var redisDb *redis.Client

// 根据redis配置初始化一个客户端
func initClient() (err error) {
	redisDb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // redis地址
		Password: "",               // redis密码，没有则留空
		DB:       0,                // 默认数据库，默认是0
	})

	//通过 *redis.Client.Ping() 来检查是否成功连接到了redis服务器
	_, err = redisDb.Ping().Result()
	if err != nil {
		return err
	}
	return nil
}
func main() {
	r := gin.Default()
	gin.SetMode(gin.DebugMode)
	initClient()
	r.GET("/seckill", func(c *gin.Context) {
		userId := c.Query("userId")
		prodId := c.Query("prodId")

		// 参数校验 用户id和产品id
		if userId == "" || prodId == "" {
			c.JSON(429, gin.H{
				"message": "params is error",
				"code":    "200",
			})
			return
		}

		// 查看秒杀是否开始，一般都是以时间判断的
		// 此处略过
		prodUserSetKey := fmt.Sprintf("sec:%s", prodId)
		prodNumKey := fmt.Sprintf("sec:%s", prodId)
		// 判断是否已经秒杀成功
		isMember, err := redisDb.SIsMember(prodUserSetKey, userId).Result()
		if err != nil {
			c.JSON(430, gin.H{
				"message": "网络错误，请重试",
				"code":    "400",
			})
			return
		}

		if isMember {
			c.JSON(200, gin.H{
				"message": "你已经秒杀成功",
				"code":    "400",
			})
			return
		}
		// 判断是否还有库存
		prodNumStr, err := redisDb.Get(prodNumKey).Result()
		log.Println(err, prodNumStr)
		if err != nil {
			c.JSON(431, gin.H{
				"message": "网络错误，请重试",
				"code":    "400",
			})
			return
		}
		var prodNum = 0
		if prodNumStr != "" {
			prodNum, err = strconv.Atoi(prodNumStr)
			if err != nil {
				c.JSON(432, gin.H{
					"message": "网络错误，请重试" + prodNumStr,
					"code":    "400",
				})
				return
			}
		}
		if prodNum <= 0 {
			c.JSON(200, gin.H{
				"message": "商品已经秒杀完成",
				"code":    "400",
			})
			return
		}
		// 减库存、增加秒杀成功数据
		redisDb.Decr(prodNumKey)
		redisDb.SAdd(prodUserSetKey, userId)
		// 返回成功
		c.JSON(200, gin.H{
			"message": "秒杀成功",
			"code":    "200",
		})
	})
	r.Run() // 监听并在 0.0.0.0:8080 上启动服务
}
