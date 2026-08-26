package model

import "time"

// ScrapeMethod 刮削方式
const (
	ScrapeMethodManual = "manual" // 手动
	ScrapeMethodAuto   = "auto"   // 自动
)

// Scrape status 刮削状态
const (
	ScrapeStatusInProgress = "in_progress" // 刮削中
	ScrapeStatusSuccess    = "success"     // 刮削成功
	ScrapeStatusFailed     = "failed"      // 刮削失败
	ScrapeStatusCompleted  = "completed"   // 刮削完成
)

// Scrape step names 刮削步骤
const (
	StepSearch           = "search"           // 搜索中
	StepGetDetail        = "get_detail"       // 获取详情
	StepDownloadPoster   = "download_poster"  // 下载封面
	StepDownloadBackdrop = "download_backdrop" // 下载背景图
	StepSearchActor       = "search_actor"     // 搜索演员
	StepTranslate         = "translate"        // 翻译中
)

// ScrapeLog 刮削日志
type ScrapeLog struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	TaskRunID  uint      `json:"task_run_id" gorm:"index"`             // 关联的 TaskRunRecord ID（手动刮削为0）
	ItemGUID   string    `json:"item_guid" gorm:"index"`              // 影片 GUID
	Title      string    `json:"title" gorm:"type:text"`              // 影片标题
	Number     string    `json:"number" gorm:"type:text"`            // 番号（metatube 搜索结果中的 number）
	Method     string    `json:"method" gorm:"type:text"`             // 刮削方式：manual / auto
	Status     string    `json:"status" gorm:"type:text;default:'in_progress'"` // 刮削状态
	Steps      string    `json:"steps" gorm:"type:text"`             // 刮削步骤记录（JSON）
	Error      string    `json:"error" gorm:"type:text"`              // 错误信息
	CreatedAt  time.Time `json:"created_at" gorm:"index"`            // 刮削时间
	UpdatedAt  time.Time `json:"updated_at"`                          // 更新时间
}

// TableName 指定表名
func (ScrapeLog) TableName() string {
	return "scrape_logs"
}
