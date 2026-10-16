package model

import "time"

// MovieSearchConfig 影片搜索配置
type MovieSearchConfig struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Source       string    `json:"source" gorm:"not null;default:javbus"`
	JavDBDomain  string    `json:"javdb_domain" gorm:"not null;default:www.javdb.com"`
	JavBusDomain string    `json:"javbus_domain" gorm:"not null;default:www.javbus.com"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
