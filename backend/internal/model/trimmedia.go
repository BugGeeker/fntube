package model

import "time"

// TrimMediaConfig 飞牛影视配置
type TrimMediaConfig struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	Host          string    `json:"host" gorm:"not null"`
	Username      string    `json:"username" gorm:"not null"`
	Password      string    `json:"password" gorm:"not null"`
	AccessCode    string    `json:"access_code"`
	PlayHost      string    `json:"play_host"`
	SyncLibraries string    `json:"sync_libraries"` // JSON array as string
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
