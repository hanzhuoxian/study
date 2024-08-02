// Package seed 处理数据库填充相关逻辑
package seed

import (
	"gohub/pkg/database"

	"gorm.io/gorm"
)

// SeederFunc SeederFunc
type SeederFunc func(*gorm.DB)

// Seeder 对应每一个 database/seeders 目录下的 Seeder 文件
type Seeder struct {
	Func SeederFunc
	Name string
}

// seeders 存放所有 seeder
var seeders []Seeder

// 按顺序执行的 Seeder 数组
var orderedSeederNames []string

// Add 注册到  seeders 数组中
func Add(name string, fn SeederFunc) {
	seeders = append(seeders, Seeder{
		Name: name,
		Func: fn,
	})
}

// SetRunOrder 设置优先运行的 seeder
func SetRunOrder(names []string) {
	orderedSeederNames = names
}

// GetSeeder 通过名称获取 Seeder 对象
func GetSeeder(name string) Seeder {
	for _, s := range seeders {
		if name == s.Name {
			return s
		}
	}
	return Seeder{}
}

// RunAll 运行所有 seeder
func RunAll() {
	executed := make(map[string]string)
	for _, name := range orderedSeederNames {
		s := GetSeeder(name)
		s.Func(database.DB)
		executed[name] = name
	}
	for _, s := range seeders {
		if _, ok := executed[s.Name]; ok {
			continue
		}
		s.Func(database.DB)
	}
}

// RunSeeder 运行单个 seeder
func RunSeeder(name string) {
	s := GetSeeder(name)
	if s.Func == nil {
		return
	}
	s.Func(database.DB)
}
