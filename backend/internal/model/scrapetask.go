package model

import "time"

// ScrapeTask 刮削计划任务
type ScrapeTask struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	Name        string     `json:"name" gorm:"type:text"`           // 任务名称
	LibraryID   string     `json:"library_id" gorm:"type:text;index"` // 媒体库 ID
	LibraryName string     `json:"library_name" gorm:"type:text"`   // 媒体库名称（冗余存储）
	Interval    int        `json:"interval" gorm:"default:60"`      // 扫描频率（分钟）
	Enabled     bool       `json:"enabled" gorm:"default:true"`    // 是否启用
	LastRunAt   *time.Time `json:"last_run_at"`                     // 上次执行时间
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// TableName 指定表名
func (ScrapeTask) TableName() string {
	return "scrape_tasks"
}
