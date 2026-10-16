package model

import "time"

// GuangyaConfig stores the bound Guangya account credentials.
// Token fields must never be exposed directly by configuration APIs.
type GuangyaConfig struct {
	ID             uint       `json:"id" gorm:"primaryKey"`
	AccessToken    string     `json:"-" gorm:"type:text;not null"`
	RefreshToken   string     `json:"-" gorm:"type:text"`
	DeviceID       string     `json:"-" gorm:"not null"`
	Subject        string     `json:"subject"`
	Name           string     `json:"name"`
	PhoneNumber    string     `json:"-" gorm:"column:phone_number"`
	LoginMethod    string     `json:"login_method"`
	TokenExpiresAt *time.Time `json:"token_expires_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (GuangyaConfig) TableName() string {
	return "guangya_configs"
}

type GuangyaCloudAddJob struct {
	ID                   uint       `json:"id" gorm:"primaryKey"`
	TaskID               string     `json:"task_id" gorm:"uniqueIndex;not null"`
	ParentID             string     `json:"parent_id"`
	RequestedName        string     `json:"requested_name" gorm:"not null"`
	ResolvedIsDirectory  bool       `json:"resolved_is_directory"`
	SelectedFiles        string     `json:"-" gorm:"type:text;not null"`
	SmartStrmTaskName    string     `json:"smartstrm_task_name" gorm:"column:smart_strm_task_name"`
	SmartStrmStorageRoot string     `json:"smartstrm_storage_root" gorm:"column:smart_strm_storage_root"`
	SmartStrmStoragePath string     `json:"smartstrm_storage_path" gorm:"column:smart_strm_storage_path"`
	StrmSavePath         string     `json:"strm_save_path"`
	SmartStrmNotified    bool       `json:"smartstrm_notified" gorm:"column:smart_strm_notified"`
	SmartStrmNotifiedAt  *time.Time `json:"smartstrm_notified_at,omitempty" gorm:"column:smart_strm_notified_at"`
	Status               string     `json:"status" gorm:"index;not null"`
	CurrentStep          string     `json:"current_step"`
	Progress             int        `json:"progress"`
	Error                string     `json:"error" gorm:"type:text"`
	CompletedAt          *time.Time `json:"completed_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

func (GuangyaCloudAddJob) TableName() string {
	return "guangya_cloud_add_jobs"
}
