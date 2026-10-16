package model

import "time"

// SmartStrmConfig stores the SmartStrm connection settings.
type SmartStrmConfig struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	URL          string    `json:"url" gorm:"column:url;not null"`
	Username     string    `json:"username"`
	Password     string    `json:"password"`
	StrmRootPath string    `json:"strm_root_path" gorm:"column:strm_root_path"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (SmartStrmConfig) TableName() string {
	return "smartstrm_configs"
}
