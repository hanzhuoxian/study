package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var redisDb redis.UniversalClient

func initClient(ctx context.Context) error {
	redisDb = redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs: []string{"localhost:6379"},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := redisDb.Ping(ctx).Result()
	return err
}

func main() {
	if err := initClient(context.Background()); err != nil {
		panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	redisDb.Set(ctx, "name", "zhuoxian", 1*time.Second)
	result, err := redisDb.Get(ctx, "name").Result()
	if err == redis.Nil {
		fmt.Println("key does not exist")
		os.Exit(0)
	}
	if err != nil {
		panic(err)
	}
	fmt.Println(result)
	ranking()
}

func ranking() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	redisDb.ZAdd(ctx, "ranking", redis.Z{
		Score:  1.0,
		Member: "zhuoxian",
	})
	redisDb.ZIncrBy(ctx, "ranking", 1.0, "zhuoxian")
	redisDb.ZAdd(ctx, "ranking", redis.Z{
		Score:  3.0,
		Member: "liting",
	})
	redisDb.ZAdd(ctx, "ranking", redis.Z{
		Score:  4.0,
		Member: "zhangsan",
	})
	c, err := redisDb.ZCard(ctx, "ranking").Result()
	if err != nil {
		panic(err)
	}
	fmt.Println(c)
	z, err := redisDb.ZRangeWithScores(ctx, "ranking", 0, 1).Result()
	if err != nil {
		panic(err)
	}

	fmt.Println(z)
	n, err := redisDb.SetNX(ctx, "name1", "zhuoxian", 1*time.Second).Result()
	if err != nil {
		panic(err)
	}
	fmt.Println(n)
	n, err = redisDb.SetNX(ctx, "name1", "zhuoxian", 1*time.Second).Result()
	if err != nil {
		panic(err)
	}
	fmt.Println(n)

	s, err := redisDb.ZRevRange(ctx, "ranking", 0, 1).Result()
	if err != nil {
		panic(err)
	}
	fmt.Println(s)

	type User struct {
		Name string
		Age  int
	}
	user := User{
		Name: "zhuoxian",
		Age:  18,
	}
	redisDb.HSet(ctx, "user:1", "Name", user.Name, "Age", user.Age)
	result, err := redisDb.HGet(ctx, "user:1", "Name").Result()
	if err != nil {
		panic(err)
	}
	fmt.Println(result)
}
