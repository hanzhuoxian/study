package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

var redisDb *redis.Client

func initClient(ctx context.Context) error {
	redisDb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	_, err := redisDb.Ping(ctx).Result()
	return err
}

func main() {
	ctx := context.Background()

	if err := initClient(ctx); err != nil {
		log.Fatalf("init redis client failed: %v", err)
	}

	for i := 1; i <= 30; i++ {
		key := fmt.Sprintf("key:%d", i)

		if err := redisDb.Set(ctx, key, i, 2*time.Minute).Err(); err != nil {
			log.Fatalf("set %s failed: %v", key, err)
		}

		value, err := redisDb.Get(ctx, key).Result()
		if err != nil {
			log.Fatalf("get %s failed: %v", key, err)
		}
		fmt.Printf("%s = %s\n", key, value)
	}
}
