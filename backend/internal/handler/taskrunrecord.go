package handler

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"gorm.io/gorm"

	"fntube/internal/model"
)

// TaskRunRecordHandler 刮削任务运行记录处理器
type TaskRunRecordHandler struct {
	db *gorm.DB
}

// NewTaskRunRecordHandler 创建刮削任务运行记录处理器
func NewTaskRunRecordHandler(db *gorm.DB) *TaskRunRecordHandler {
	return &TaskRunRecordHandler{db: db}
}

// RegisterTaskRunRecordHandlers 注册刮削任务运行记录相关路由
func RegisterTaskRunRecordHandlers(h *server.Hertz, db *gorm.DB) {
	hd := NewTaskRunRecordHandler(db)
	g := h.Group("/api/taskrun")
	g.GET("/list", hd.list)
	g.GET("/detail/:id", hd.detail)
}

// list 获取刮削任务运行记录列表（分页）
func (h *TaskRunRecordHandler) list(ctx context.Context, c *app.RequestContext) {
	page, _ := strconv.Atoi(string(c.Query("page")))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(string(c.Query("page_size")))
	if pageSize <= 0 {
		pageSize = 20
	}

	var total int64
	if err := h.db.Model(&model.TaskRunRecord{}).Count(&total).Error; err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	var records []model.TaskRunRecord
	if err := h.db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records).Error; err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{
		"total": total,
		"items": records,
	})
}

// detail 获取某次运行的刮削日志明细
func (h *TaskRunRecordHandler) detail(ctx context.Context, c *app.RequestContext) {
	idStr := string(c.Param("id"))
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.JSON(400, map[string]string{"error": "无效的 id"})
		return
	}

	var record model.TaskRunRecord
	if err := h.db.First(&record, id).Error; err != nil {
		c.JSON(404, map[string]string{"error": "记录不存在"})
		return
	}

	var logs []model.ScrapeLog
	if err := h.db.Where("task_run_id = ?", id).Order("id desc").Find(&logs).Error; err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{
		"record": record,
		"logs":   logs,
	})
}
