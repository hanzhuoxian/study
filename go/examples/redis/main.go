package main

import (
	"fmt"
	"log"
	"time"

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
	err := initClient()
	if err != nil {
		log.Fatal("init redis client failed!")
	}
	for i := 1; i <= 30; i++ {
		key := fmt.Sprintf("key:%d", i)
		statusCmd := redisDb.Set(key, i, 2*time.Minute)
		if nil != statusCmd.Err() {
			panic(err)
		}
		fmt.Println(statusCmd)

		getStatus := redisDb.Get(key)
		value, err := getStatus.Result()
		if nil != err {
			panic(err)
		}
		fmt.Println(value)
	}
}
