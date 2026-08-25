package handler

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"gorm.io/gorm"

	"fntube/internal/model"
	"fntube/internal/scheduler"
	"fntube/internal/trimmedia"
)

// ScrapeTaskHandler 刮削计划任务处理器
type ScrapeTaskHandler struct {
	db        *gorm.DB
	service   *trimmedia.Service
	scheduler *scheduler.Scheduler
}

// NewScrapeTaskHandler 创建刮削计划任务处理器
func NewScrapeTaskHandler(db *gorm.DB, service *trimmedia.Service, sched *scheduler.Scheduler) *ScrapeTaskHandler {
	return &ScrapeTaskHandler{db: db, service: service, scheduler: sched}
}

// RegisterScrapeTaskHandlers 注册刮削计划任务相关路由
func RegisterScrapeTaskHandlers(h *server.Hertz, db *gorm.DB, service *trimmedia.Service, sched *scheduler.Scheduler) {
	hd := NewScrapeTaskHandler(db, service, sched)
	g := h.Group("/api/scrapetask")
	g.GET("/list", hd.list)
	g.POST("/create", hd.create)
	g.POST("/update", hd.update)
	g.DELETE("/:id", hd.delete)
	g.POST("/run/:id", hd.runNow)
}

// list 获取刮削计划任务列表
func (h *ScrapeTaskHandler) list(ctx context.Context, c *app.RequestContext) {
	var tasks []model.ScrapeTask
	if err := h.db.Order("id desc").Find(&tasks).Error; err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, tasks)
}

// create 创建刮削计划任务
func (h *ScrapeTaskHandler) create(ctx context.Context, c *app.RequestContext) {
	var req struct {
		Name        string `json:"name"`
		LibraryID   string `json:"library_id"`
		LibraryName string `json:"library_name"`
		Interval    int    `json:"interval"`
		Enabled     bool   `json:"enabled"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}
	if req.Name == "" {
		c.JSON(400, map[string]string{"error": "name 不能为空"})
		return
	}
	if req.LibraryID == "" {
		c.JSON(400, map[string]string{"error": "library_id 不能为空"})
		return
	}
	if req.Interval <= 0 {
		req.Interval = 60
	}

	task := model.ScrapeTask{
		Name:        req.Name,
		LibraryID:   req.LibraryID,
		LibraryName: req.LibraryName,
		Interval:    req.Interval,
		Enabled:     req.Enabled,
	}
	if err := h.db.Create(&task).Error; err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, task)
}

// update 更新刮削计划任务
func (h *ScrapeTaskHandler) update(ctx context.Context, c *app.RequestContext) {
	var req struct {
		ID          uint   `json:"id"`
		Name        string `json:"name"`
		LibraryID   string `json:"library_id"`
		LibraryName string `json:"library_name"`
		Interval    int    `json:"interval"`
		Enabled     bool   `json:"enabled"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}
	if req.ID == 0 {
		c.JSON(400, map[string]string{"error": "id 不能为空"})
		return
	}

	updates := map[string]interface{}{
		"name":         req.Name,
		"library_id":   req.LibraryID,
		"library_name": req.LibraryName,
		"interval":     req.Interval,
		"enabled":      req.Enabled,
	}
	if err := h.db.Model(&model.ScrapeTask{}).Where("id = ?", req.ID).Updates(updates).Error; err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	var task model.ScrapeTask
	h.db.First(&task, req.ID)
	c.JSON(200, task)
}

// delete 删除刮削计划任务
func (h *ScrapeTaskHandler) delete(ctx context.Context, c *app.RequestContext) {
	idStr := string(c.Param("id"))
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.JSON(400, map[string]string{"error": "无效的 id"})
		return
	}

	if err := h.db.Delete(&model.ScrapeTask{}, id).Error; err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, map[string]string{"status": "ok"})
}

// runNow 立即执行刮削计划任务
func (h *ScrapeTaskHandler) runNow(ctx context.Context, c *app.RequestContext) {
	idStr := string(c.Param("id"))
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.JSON(400, map[string]string{"error": "无效的 id"})
		return
	}

	// 异步执行，避免请求超时
	go func() {
		if err := h.scheduler.RunTask(uint(id)); err != nil {
			// 日志记录在 scheduler 内部
		}
	}()

	c.JSON(200, map[string]string{"status": "ok", "message": "任务已开始执行"})
}
