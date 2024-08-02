package verifycode

import (
	"gohub/pkg/app"
	"gohub/pkg/config"
	"gohub/pkg/redis"
	"time"
)

type Store interface {
	// 保存验证码
	Set(id string, value string) bool
	// 获取验证码
	Get(id string, clear bool) string
	// 检查验证码
	Verify(id, answer string, clear bool) bool
}

// RedisStore 实现 verifycode.Store interface
type RedisStore struct {
	RedisClient *redis.RedisClient
	KeyPrefix   string
}

// Set 保存验证码
func (s *RedisStore) Set(key string, value string) bool {
	ExpireTime := time.Minute * time.Duration(config.GetInt64("verifycode.expire_time"))
	if app.IsLocal() {
		ExpireTime = time.Minute * time.Duration(config.GetInt64("verifycode.debug_expire_time"))
	}
	return s.RedisClient.Set(s.KeyPrefix+key, value, ExpireTime)
}

// Get 获取验证码
func (s *RedisStore) Get(key string, clear bool) string {
	code := s.RedisClient.Get(s.KeyPrefix + key)
	if clear {
		s.RedisClient.Del(s.KeyPrefix + key)
	}
	return code
}

// Verify 验证验证码
func (s *RedisStore) Verify(id, answer string, clear bool) bool {
	v := s.Get(id, clear)
	return v == answer
}
