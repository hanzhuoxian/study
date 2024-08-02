// Package reeis 工具包
package redis

import (
	"context"
	"gohub/pkg/logger"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient Redis服务
type RedisClient struct {
	Client  *redis.Client
	Context context.Context
}

// once 确保全局的Redis对象只实例一次
var once sync.Once

// Redis 全局 Redis
var Redis *RedisClient

// Connect 连接Redis
func Connect(address string, username string, password string, db int) {
	once.Do(func() {
		Redis = NewClient(address, username, password, db)
	})
}

// NewClient 创建Redis客户端
func NewClient(address string, username string, password string, db int) *RedisClient {
	// 初始化实例
	r := &RedisClient{}
	// 使用默认的 context
	r.Context = context.Background()
	// 使用 redis 库里的 NewClient 初始化连接
	r.Client = redis.NewClient(&redis.Options{
		Addr:     address,
		Username: username,
		Password: password,
		DB:       db,
	})
	// 测试下连接
	err := r.Ping()
	logger.LogIf(err)
	return r
}

// Ping 用以测试 redis 连接是否正常
func (r RedisClient) Ping() error {
	_, err := r.Client.Ping(r.Context).Result()
	return err
}

// Set 存储 key 对应的 value ，且设置expiration过期时间
func (r RedisClient) Set(key string, value interface{}, expiration time.Duration) bool {
	if err := r.Client.Set(r.Context, key, value, expiration).Err(); err != nil {
		logger.ErrorString("Redis", "Set", err.Error())
		return false
	}
	return true
}

// Get 获取 key 对应的value
func (r RedisClient) Get(key string) string {
	result, err := r.Client.Get(r.Context, key).Result()
	if err != nil {
		if err != redis.Nil {
			logger.ErrorString("Redis", "Get", err.Error())
		}
		return ""
	}
	return result
}

// Has 判断一个 key 是否存在，内部错误和 redis.Nil 都返回 false
func (r RedisClient) Has(key string) bool {
	_, err := r.Client.Get(r.Context, key).Result()
	if err != nil {
		if err != redis.Nil {
			logger.ErrorString("Redis", "Get", err.Error())
		}
		return false
	}
	return true
}

// Del 删除存储在 redis 中的数据，支持多个key
func (r RedisClient) Del(keys ...string) bool {
	if err := r.Client.Del(r.Context, keys...).Err(); err != nil {
		logger.ErrorString("Redis", "Del", err.Error())
		return false
	}
	return true
}

// FlushDB 清空当前 redis 里的所有数据
func (r RedisClient) FlushDB() bool {
	if err := r.Client.FlushDB(r.Context).Err(); err != nil {
		logger.ErrorString("Redis", "FlushDB", err.Error())
		return false
	}
	return true
}

// Decrement 当参数只有一个时为 key 的值 加 1
// 当参数有两个时，每一个参数发key，第二个参数为要加去的 int64 类型
func (r RedisClient) Increment(parameters ...interface{}) bool {
	switch len(parameters) {
	case 1:
		key := parameters[0].(string)
		if err := r.Client.Incr(r.Context, key).Err(); err != nil {
			logger.ErrorString("Redis", "Increment", err.Error())
			return false
		}

	case 2:
		key := parameters[0].(string)
		value := parameters[1].(int64)
		if err := r.Client.IncrBy(r.Context, key, value).Err(); err != nil {
			logger.ErrorString("Redis", "Increment", err.Error())
			return false
		}
	default:
		logger.ErrorString("Redis", "Increment", "参数过多")
		return false
	}
	return true
}

// Decrement 当参数只有一个时为 key 的值 减 1
// 当参数有两个时，每一个参数发key，第二个参数为要减去的 int64 类型
func (r RedisClient) Decrement(parameters ...interface{}) bool {
	switch len(parameters) {
	case 1:
		key := parameters[0].(string)
		if err := r.Client.Decr(r.Context, key).Err(); err != nil {
			logger.ErrorString("Redis", "Increment", err.Error())
			return false
		}

	case 2:
		key := parameters[0].(string)
		value := parameters[1].(int64)
		if err := r.Client.DecrBy(r.Context, key, value).Err(); err != nil {
			logger.ErrorString("Redis", "Increment", err.Error())
			return false
		}
	default:
		logger.ErrorString("Redis", "Increment", "参数过多")
		return false
	}
	return true
}
