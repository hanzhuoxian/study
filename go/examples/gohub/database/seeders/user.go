// Package seeders 数据填充
package seeders

import (
	"fmt"
	"gohub/database/factories"
	"gohub/pkg/console"
	"gohub/pkg/logger"
	"gohub/pkg/seed"

	"gorm.io/gorm"
)

func init() {
	seed.Add("user", func(db *gorm.DB) {
		users := factories.MakeUser(10)
		result := db.Table("users").Create(&users)
		if result.Error != nil {
			logger.LogIf(result.Error)
			return
		}

		console.Success(fmt.Sprintf("Table [%v] %v rows seeded", result.Statement.Table, result.RowsAffected))
	})
}
