package model

import "time"

// ScrapeMethod 刮削方式
const (
	ScrapeMethodManual = "manual" // 手动
	ScrapeMethodAuto   = "auto"   // 自动
)

// ScrapeLog 刮削日志
type ScrapeLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	ItemGUID  string    `json:"item_guid" gorm:"index"`         // 影片 GUID
	Title     string    `json:"title" gorm:"type:text"`         // 影片标题
	Number    string    `json:"number" gorm:"type:text"`        // 番号（metatube 搜索结果中的 number）
	Method    string    `json:"method" gorm:"type:text"`        // 刮削方式：manual / auto
	CreatedAt time.Time `json:"created_at" gorm:"index"`        // 刮削时间
}

// TableName 指定表名
func (ScrapeLog) TableName() string {
	return "scrape_logs"
}
