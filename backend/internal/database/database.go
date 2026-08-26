package database

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"fntube/internal/model"
)

// Init 初始化 SQLite 数据库并自动迁移表结构
// 使用 glebarez/sqlite 纯 Go 驱动，无需 CGO 即可交叉编译
func Init(dbPath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	// 自动迁移
	if err := db.AutoMigrate(&model.TrimMediaConfig{}, &model.MetaTubeConfig{}, &model.ScrapeLog{}, &model.ScrapeTask{}, &model.TaskRunRecord{}); err != nil {
		return nil, err
	}

	return db, nil
}
