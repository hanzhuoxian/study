package config

import "gohub/pkg/config"

func init() {
	config.Add("log", func() map[string]interface{} {
		return map[string]interface{}{
			// 日志级别，必须是以下这些选项
			// debug -- 信息量大、一般调试时打开。系统模块详细运行的日志
			// info -- 业务级别的运行日志，如用户登录、用户退出、订单撤销
			// warn -- 感兴趣需要引起关注
			// error -- 记录错误信息， Panic 或者 Error
			"level": config.Env("LOG_LEVEL", "debug"),

			// 日志类型
			// single 独立的文件
			// daily 按照日期每日一个
			"type": config.Env("LOG_TYPE", "single"),
			// 日志文件路径
			"filename": config.Env("LOG_NAME", "storage/logs/logs.log"),
			// 每个日志文件保存的最大值 单位: M
			"max_size": config.Env("LOG_MAX_SIZE", 64),
			// 最多保存的日志文件数,0为不限， maxage 到了还是会删
			"max_backup": config.Env("LOG_MAX_BACKUP", 5),
			// 最多保存多少天 7 表示一周前的日志会被删除 0 表示不删除
			"max_age": config.Env("LOG_MAX_AGE", 30),
			// 是否压缩，压缩日志不方便查看
			"compress": config.Env("LOG_COMPRESS", false),
		}
	})
}
