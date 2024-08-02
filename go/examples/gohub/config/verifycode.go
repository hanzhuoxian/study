package config

import "gohub/pkg/config"

func init() {
	config.Add("verifycode", func() map[string]interface{} {
		return map[string]interface{}{
			"code_length":       config.Env("VERIFY_CODE", 6),
			"expire_time":       config.Env("VERIFY_EXPIRE_TIME", 15),
			"debug_expire_time": 10080,
			"debug_code":        "123456",
		}
	})
}
