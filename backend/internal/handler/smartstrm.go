package handler

import (
	"context"
	"net/url"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"gorm.io/gorm"

	"fntube/internal/smartstrm"
)

type SmartStrmHandler struct {
	db *gorm.DB
}

type smartStrmConfigView struct {
	ID           uint   `json:"id"`
	URL          string `json:"url"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	StrmRootPath string `json:"strm_root_path"`
}

type smartStrmVersion = smartstrm.Version
type smartStrmTask = smartstrm.Task

func RegisterSmartStrmHandlers(h *server.Hertz, db *gorm.DB) {
	handler := &SmartStrmHandler{db: db}
	group := h.Group("/api/smartstrm")
	group.GET("/config", handler.getConfig)
	group.POST("/config", handler.saveConfig)
	group.POST("/test", handler.testConnection)
	group.GET("/tasks", handler.getTasks)
}

func (h *SmartStrmHandler) getConfig(ctx context.Context, c *app.RequestContext) {
	var config smartStrmConfigView
	_ = h.db.Table("smartstrm_configs").Order("id desc").First(&config).Error
	c.JSON(200, config)
}

func (h *SmartStrmHandler) saveConfig(ctx context.Context, c *app.RequestContext) {
	var request smartStrmConfigView
	if err := c.BindJSON(&request); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}

	request.URL = strings.TrimRight(strings.TrimSpace(request.URL), "/")
	request.Username = strings.TrimSpace(request.Username)
	request.StrmRootPath = strings.TrimSpace(request.StrmRootPath)
	if request.URL == "" {
		c.JSON(400, map[string]string{"error": "请填写 SmartStrm 地址"})
		return
	}
	parsed, err := url.Parse(request.URL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") {
		c.JSON(400, map[string]string{"error": "SmartStrm 地址必须是 http 或 https URL"})
		return
	}
	if request.Username == "" {
		c.JSON(400, map[string]string{"error": "请填写 SmartStrm 账号"})
		return
	}
	if request.Password == "" {
		c.JSON(400, map[string]string{"error": "请填写 SmartStrm 密码"})
		return
	}
	if request.StrmRootPath == "" {
		c.JSON(400, map[string]string{"error": "请填写 strm 根目录"})
		return
	}

	values := map[string]interface{}{
		"url":            request.URL,
		"username":       request.Username,
		"password":       request.Password,
		"strm_root_path": request.StrmRootPath,
	}
	var id uint
	_ = h.db.Table("smartstrm_configs").Select("id").Order("id desc").Limit(1).Row().Scan(&id)
	if id > 0 {
		err = h.db.Table("smartstrm_configs").Where("id = ?", id).Updates(values).Error
	} else {
		err = h.db.Table("smartstrm_configs").Create(values).Error
	}
	if err != nil {
		c.JSON(500, map[string]string{"error": "保存 SmartStrm 配置失败: " + err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{"status": "ok", "config": request})
}

func (h *SmartStrmHandler) testConnection(ctx context.Context, c *app.RequestContext) {
	var request smartStrmConfigView
	if err := c.BindJSON(&request); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}

	request.URL = strings.TrimRight(strings.TrimSpace(request.URL), "/")
	request.Username = strings.TrimSpace(request.Username)
	if request.URL == "" || request.Username == "" || request.Password == "" {
		c.JSON(400, map[string]string{"error": "请填写 SmartStrm 地址、账号和密码"})
		return
	}
	parsed, err := url.Parse(request.URL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") {
		c.JSON(400, map[string]string{"error": "SmartStrm 地址必须是 http 或 https URL"})
		return
	}

	version, err := testSmartStrmConnection(ctx, request.URL, request.Username, request.Password)
	if err != nil {
		c.JSON(400, map[string]string{"error": "连接失败: " + err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{
		"status":            "ok",
		"version":           version.Version,
		"remote_version":    version.RemoteVersion,
		"is_new":            version.IsNew,
		"notify":            version.Notify,
		"license_store_url": version.LicenseStoreURL,
		"change_log_url":    version.ChangeLogURL,
	})
}

func (h *SmartStrmHandler) getTasks(ctx context.Context, c *app.RequestContext) {
	var config smartStrmConfigView
	if err := h.db.Table("smartstrm_configs").Order("id desc").First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(400, map[string]string{"error": "请先保存 SmartStrm 配置"})
			return
		}
		c.JSON(500, map[string]string{"error": "读取 SmartStrm 配置失败: " + err.Error()})
		return
	}

	tasks, err := fetchSmartStrmTasks(ctx, config.URL, config.Username, config.Password)
	if err != nil {
		c.JSON(400, map[string]string{"error": "读取 SmartStrm 任务失败: " + err.Error()})
		return
	}
	c.JSON(200, tasks)
}

func testSmartStrmConnection(
	ctx context.Context,
	baseURL string,
	username string,
	password string,
) (smartStrmVersion, error) {
	return smartstrm.TestConnection(ctx, baseURL, username, password)
}

func fetchSmartStrmTasks(
	ctx context.Context,
	baseURL string,
	username string,
	password string,
) (map[string]smartStrmTask, error) {
	return smartstrm.FetchTasks(ctx, baseURL, username, password)
}
