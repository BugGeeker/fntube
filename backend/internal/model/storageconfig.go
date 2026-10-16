package model

import "time"

// StorageMapping stores a Guangya-to-SmartStrm storage mapping.
type StorageMapping struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	Name              string    `json:"name" gorm:"not null"`
	GuangyaFolderID   string    `json:"guangya_folder_id"`
	GuangyaFolderPath string    `json:"guangya_folder_path" gorm:"not null"`
	SmartStrmTaskName string    `json:"smartstrm_task_name" gorm:"not null"`
	StrmSavePath      string    `json:"strm_save_path" gorm:"not null"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (StorageMapping) TableName() string {
	return "storage_mappings"
}
