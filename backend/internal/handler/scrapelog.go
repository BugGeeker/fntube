package handler

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"gorm.io/gorm"

	"fntube/internal/model"
	"fntube/internal/scheduler"
)

// ScrapeLogHandler 刮削日志处理器
type ScrapeLogHandler struct {
	db        *gorm.DB
	scheduler *scheduler.Scheduler
}

// NewScrapeLogHandler 创建刮削日志处理器
func NewScrapeLogHandler(db *gorm.DB, sched *scheduler.Scheduler) *ScrapeLogHandler {
	return &ScrapeLogHandler{db: db, scheduler: sched}
}

// RegisterScrapeLogHandlers 注册刮削日志相关路由
func RegisterScrapeLogHandlers(h *server.Hertz, db *gorm.DB, sched *scheduler.Scheduler) {
	hd := NewScrapeLogHandler(db, sched)
	g := h.Group("/api/scrapelog")
	g.GET("/list", hd.list)
	g.POST("/create", hd.create)
	g.DELETE("/:id", hd.delete)
	g.POST("/rescrape/:itemGuid", hd.rescrape)
}

// list 获取刮削日志列表（分页）
func (h *ScrapeLogHandler) list(ctx context.Context, c *app.RequestContext) {
	page, _ := strconv.Atoi(string(c.Query("page")))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(string(c.Query("page_size")))
	if pageSize <= 0 {
		pageSize = 20
	}

	var total int64
	if err := h.db.Model(&model.ScrapeLog{}).Count(&total).Error; err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	var logs []model.ScrapeLog
	if err := h.db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error; err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{
		"total": total,
		"items": logs,
	})
}

// create 创建刮削日志
func (h *ScrapeLogHandler) create(ctx context.Context, c *app.RequestContext) {
	var req struct {
		ItemGUID string `json:"item_guid"`
		Title    string `json:"title"`
		Number   string `json:"number"`
		Method   string `json:"method"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}
	if req.ItemGUID == "" {
		c.JSON(400, map[string]string{"error": "item_guid 不能为空"})
		return
	}
	if req.Method == "" {
		req.Method = model.ScrapeMethodManual
	}

	log := model.ScrapeLog{
		ItemGUID: req.ItemGUID,
		Title:    req.Title,
		Number:   req.Number,
		Method:   req.Method,
	}
	if err := h.db.Create(&log).Error; err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, log)
}

// delete 删除刮削日志
func (h *ScrapeLogHandler) delete(ctx context.Context, c *app.RequestContext) {
	id := string(c.Param("id"))
	if id == "" {
		c.JSON(400, map[string]string{"error": "id 不能为空"})
		return
	}

	if err := h.db.Delete(&model.ScrapeLog{}, id).Error; err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, map[string]string{"status": "ok"})
}

// rescrape 重新刮削指定媒体项
func (h *ScrapeLogHandler) rescrape(ctx context.Context, c *app.RequestContext) {
	itemGUID := string(c.Param("itemGuid"))
	if itemGUID == "" {
		c.JSON(400, map[string]string{"error": "itemGuid 不能为空"})
		return
	}

	// 先删除该 item_guid 的旧日志
	h.db.Where("item_guid = ?", itemGUID).Delete(&model.ScrapeLog{})

	// 异步执行刮削
	go func() {
		if _, err := h.scheduler.ScrapeSingle(itemGUID); err != nil {
			// 错误日志在 scheduler 内部记录
		}
	}()

	c.JSON(200, map[string]string{"status": "ok", "message": "刮削已开始"})
}
