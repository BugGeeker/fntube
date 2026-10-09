package handler

import (
	"context"
	"log"
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

	query := h.db.Model(&model.ScrapeLog{})
	if number := string(c.Query("number")); number != "" {
		query = query.Where("number LIKE ?", "%"+number+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	var logs []model.ScrapeLog
	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error; err != nil {
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

	// 先创建刮削记录，再异步执行，确保前置检查失败时也能看到失败原因。
	if err := h.scheduler.StartScrapeSingle(itemGUID); err != nil {
		log.Printf("[scrapelog] 启动单条刮削失败 %s: %v", itemGUID, err)
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, map[string]string{"status": "ok", "message": "刮削已开始"})
}
