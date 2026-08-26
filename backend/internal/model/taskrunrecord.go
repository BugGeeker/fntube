package model

import "time"

// TaskRunRecord 刮削任务运行记录
type TaskRunRecord struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	TaskID      uint      `json:"task_id" gorm:"index"`                // 关联的 ScrapeTask ID
	TaskName    string    `json:"task_name" gorm:"type:text"`          // 任务名称（冗余）
	LibraryName string    `json:"library_name" gorm:"type:text"`       // 媒体库名称（冗余）
	StartTime   time.Time `json:"start_time" gorm:"index"`            // 开始时间
	EndTime     *time.Time `json:"end_time"`                           // 结束时间
	Duration    int64     `json:"duration" gorm:"default:0"`           // 运行时长（秒）
	SuccessCount int      `json:"success_count" gorm:"default:0"`     // 成功数量（status=success）
	CompletedCount int    `json:"completed_count" gorm:"default:0"`   // 完成数量（status=completed）
	FailedCount int       `json:"failed_count" gorm:"default:0"`      // 失败数量（status=failed）
	Status      string    `json:"status" gorm:"type:text;default:'running'"` // running / done / error
	Error       string    `json:"error" gorm:"type:text"`              // 任务级错误信息
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名
func (TaskRunRecord) TableName() string {
	return "task_run_records"
}
