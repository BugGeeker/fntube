package model

import "time"

// TranslateMode 翻译模式
const (
	TranslateModeNone          = "none"           // 不翻译
	TranslateModeTitle         = "title"          // 仅标题
	TranslateModeSummary       = "summary"        // 仅简介
	TranslateModeTitleAndSummary = "title_and_summary" // 标题和简介
)

// TranslateEngine 翻译引擎
const (
	EngineBaidu      = "baidu"      // 百度翻译
	EngineDeepL      = "deepl"      // DeepL
	EngineGoogle     = "google"     // Google 翻译
	EngineGoogleFree = "googlefree" // Google Free
	EngineOpenAI     = "openai"     // OpenAI
)

// MetaTubeConfig MetaTube 配置
type MetaTubeConfig struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	Host           string    `json:"host" gorm:"not null"`             // 服务地址
	Token          string    `json:"token"`                            // API Token（非必填）
	TranslateMode  string    `json:"translate_mode" gorm:"default:none"` // 翻译模式
	TranslateEngine string   `json:"translate_engine" gorm:"default:baidu"` // 翻译引擎
	EngineConfig   string    `json:"engine_config"`                    // 引擎专属配置（JSON）
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TableName 指定表名
func (MetaTubeConfig) TableName() string {
	return "meta_tube_configs"
}