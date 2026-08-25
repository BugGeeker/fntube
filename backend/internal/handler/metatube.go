package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"gorm.io/gorm"

	"fntube/internal/metatube"
)

// MetaTubeHandler MetaTube 处理器
type MetaTubeHandler struct {
	db *gorm.DB
}

// NewMetaTubeHandler 创建 MetaTube 处理器
func NewMetaTubeHandler(db *gorm.DB) *MetaTubeHandler {
	return &MetaTubeHandler{db: db}
}

// RegisterMetaTubeHandlers 注册 MetaTube 相关路由
func RegisterMetaTubeHandlers(h *server.Hertz, db *gorm.DB) {
	hd := NewMetaTubeHandler(db)
	g := h.Group("/api/metatube")
	g.GET("/config", hd.getConfig)
	g.POST("/config", hd.saveConfig)
	g.POST("/test", hd.testConnection)
	g.GET("/search", hd.searchMovies)
	g.GET("/movie/:provider/:id", hd.getMovieInfo)
	g.GET("/translate", hd.translateText)
}

// metaTubeConfigView 配置视图，避免暴露 Token 详情
type metaTubeConfigView struct {
	ID              uint   `json:"id"`
	Host            string `json:"host"`
	Token           string `json:"token"`
	TranslateMode   string `json:"translate_mode"`
	TranslateEngine string `json:"translate_engine"`
	EngineConfig    string `json:"engine_config"`
}

// getConfig 获取 MetaTube 配置
func (h *MetaTubeHandler) getConfig(ctx context.Context, c *app.RequestContext) {
	var cfg metaTubeConfigView
	_ = h.db.Table("meta_tube_configs").Order("id desc").First(&cfg).Error
	c.JSON(200, cfg)
}

// saveConfig 保存 MetaTube 配置
func (h *MetaTubeHandler) saveConfig(ctx context.Context, c *app.RequestContext) {
	var req struct {
		Host            string `json:"host"`
		Token           string `json:"token"`
		TranslateMode   string `json:"translate_mode"`
		TranslateEngine string `json:"translate_engine"`
		EngineConfig    string `json:"engine_config"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}

	// 校验翻译模式
	if req.TranslateMode != "" && req.TranslateMode != "none" &&
		req.TranslateMode != "title" && req.TranslateMode != "summary" &&
		req.TranslateMode != "title_and_summary" {
		c.JSON(400, map[string]string{"error": "无效的翻译模式"})
		return
	}

	// 校验翻译引擎
	if req.TranslateEngine != "" && req.TranslateEngine != "baidu" &&
		req.TranslateEngine != "deepl" && req.TranslateEngine != "google" &&
		req.TranslateEngine != "googlefree" && req.TranslateEngine != "openai" {
		c.JSON(400, map[string]string{"error": "无效的翻译引擎"})
		return
	}

	// 使用原生 SQL upsert
	var id int64
	row := h.db.Table("meta_tube_configs").Select("id").Order("id desc").Limit(1).Row()
	_ = row.Scan(&id)

	values := map[string]interface{}{
		"host":             req.Host,
		"token":            req.Token,
		"translate_mode":   req.TranslateMode,
		"translate_engine": req.TranslateEngine,
		"engine_config":    req.EngineConfig,
	}

	if id > 0 {
		if err := h.db.Table("meta_tube_configs").Where("id = ?", id).Updates(values).Error; err != nil {
			c.JSON(500, map[string]string{"error": err.Error()})
			return
		}
	} else {
		if err := h.db.Table("meta_tube_configs").Create(values).Error; err != nil {
			c.JSON(500, map[string]string{"error": err.Error()})
			return
		}
	}

	c.JSON(200, map[string]string{"status": "ok"})
}

// testConnection 测试连接
// 请求 MetaTube 服务端首页版本信息（GET /），并在配置了 token 时校验 token 有效性
func (h *MetaTubeHandler) testConnection(ctx context.Context, c *app.RequestContext) {
	var req struct {
		Host  string `json:"host"`
		Token string `json:"token"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}
	if req.Host == "" {
		c.JSON(400, map[string]string{"error": "请填写服务地址"})
		return
	}

	client := metatube.NewClient(req.Host, req.Token)
	defer client.Close()

	version, err := client.Version()
	if err != nil {
		c.JSON(400, map[string]string{"error": "连接失败: " + err.Error()})
		return
	}

	// 配置了 token 时校验其有效性
	tokenValid := false
	if req.Token != "" {
		if !client.VerifyToken() {
			c.JSON(400, map[string]string{"error": "服务端可达，但 Token 验证失败，请检查 Token 是否正确"})
			return
		}
		tokenValid = true
	}

	c.JSON(200, map[string]interface{}{
		"status":      "ok",
		"app":         version.App,
		"version":     version.Version,
		"token_valid": tokenValid,
	})
}

// searchMovies 搜索影片
// 从数据库读取 MetaTube 配置，调用 MetaTube 服务端 GET /v1/movies/search?q=<keyword>
func (h *MetaTubeHandler) searchMovies(ctx context.Context, c *app.RequestContext) {
	q := string(c.Query("q"))
	if q == "" {
		c.JSON(400, map[string]string{"error": "搜索关键词不能为空"})
		return
	}

	// 从数据库读取配置
	var cfg metaTubeConfigView
	if err := h.db.Table("meta_tube_configs").Order("id desc").First(&cfg).Error; err != nil {
		c.JSON(400, map[string]string{"error": "未配置 MetaTube，请先保存配置"})
		return
	}
	if cfg.Host == "" {
		c.JSON(400, map[string]string{"error": "MetaTube 服务地址为空，请先配置"})
		return
	}

	client := metatube.NewClient(cfg.Host, cfg.Token)
	defer client.Close()

	results, err := client.SearchMovies(q)
	if err != nil {
		c.JSON(400, map[string]string{"error": "搜索失败: " + err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{
		"data": results,
	})
}

// getMovieInfo 获取影片详情
// 从数据库读取 MetaTube 配置，调用 MetaTube 服务端 GET /v1/movies/{provider}/{id}
func (h *MetaTubeHandler) getMovieInfo(ctx context.Context, c *app.RequestContext) {
	provider := string(c.Param("provider"))
	id := string(c.Param("id"))
	if provider == "" || id == "" {
		c.JSON(400, map[string]string{"error": "provider 和 id 不能为空"})
		return
	}

	// 从数据库读取配置
	var cfg metaTubeConfigView
	if err := h.db.Table("meta_tube_configs").Order("id desc").First(&cfg).Error; err != nil {
		c.JSON(400, map[string]string{"error": "未配置 MetaTube，请先保存配置"})
		return
	}
	if cfg.Host == "" {
		c.JSON(400, map[string]string{"error": "MetaTube 服务地址为空，请先配置"})
		return
	}

	client := metatube.NewClient(cfg.Host, cfg.Token)
	defer client.Close()

	info, err := client.GetMovieInfo(provider, id)
	if err != nil {
		c.JSON(400, map[string]string{"error": "获取影片详情失败: " + err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{
		"data": info,
	})
}

// translateText 文本翻译
// 从数据库读取 MetaTube 配置，调用 MetaTube 服务端 GET /v1/translate
func (h *MetaTubeHandler) translateText(ctx context.Context, c *app.RequestContext) {
	q := string(c.Query("q"))
	if q == "" {
		c.JSON(400, map[string]string{"error": "翻译文本不能为空"})
		return
	}

	// 从数据库读取配置
	var cfg metaTubeConfigView
	if err := h.db.Table("meta_tube_configs").Order("id desc").First(&cfg).Error; err != nil {
		c.JSON(400, map[string]string{"error": "未配置 MetaTube，请先保存配置"})
		return
	}
	if cfg.Host == "" {
		c.JSON(400, map[string]string{"error": "MetaTube 服务地址为空，请先配置"})
		return
	}
	if cfg.TranslateMode == "none" || cfg.TranslateMode == "" {
		c.JSON(400, map[string]string{"error": "翻译未开启"})
		return
	}

	to := string(c.Query("to"))
	if to == "" {
		to = "zh-CN"
	}

	client := metatube.NewClient(cfg.Host, cfg.Token)
	defer client.Close()

	result, err := client.Translate(q, "", to, cfg.TranslateEngine, cfg.EngineConfig)
	if err != nil {
		c.JSON(400, map[string]string{"error": "翻译失败: " + err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{
		"data": result,
	})
}