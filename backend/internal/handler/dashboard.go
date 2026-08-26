package handler

import (
	"context"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"gorm.io/gorm"

	"fntube/internal/model"
	"fntube/internal/trimmedia"
)

// DashboardHandler 总览页处理器
type DashboardHandler struct {
	db      *gorm.DB
	service *trimmedia.Service
}

// NewDashboardHandler 创建总览页处理器
func NewDashboardHandler(db *gorm.DB, service *trimmedia.Service) *DashboardHandler {
	return &DashboardHandler{db: db, service: service}
}

// RegisterDashboardHandlers 注册总览页相关路由
func RegisterDashboardHandlers(h *server.Hertz, db *gorm.DB, service *trimmedia.Service) {
	hd := NewDashboardHandler(db, service)
	g := h.Group("/api/dashboard")
	g.GET("/summary", hd.summary)
}

// dashboardDailyTrend 每日刮削数量
type dashboardDailyTrend struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// dashboardSummary 总览页汇总数据
type dashboardSummary struct {
	TotalMedia      int                    `json:"total_media"`
	WeeklyNewMedia  int                    `json:"weekly_new_media"`
	TotalScrapes    int64                  `json:"total_scrapes"`
	WeeklyScrapes   int64                  `json:"weekly_scrapes"`
	DailyScrapes    []dashboardDailyTrend  `json:"daily_scrapes"`
	TaskSummary     []dashboardTaskStatus  `json:"task_summary"`
}

// dashboardTaskStatus 刮削任务状态
type dashboardTaskStatus struct {
	ID           uint       `json:"id"`
	Name         string     `json:"name"`
	LibraryName  string     `json:"library_name"`
	Enabled      bool       `json:"enabled"`
	IsRunning    bool       `json:"is_running"`
	Interval     int        `json:"interval"`
	LastRunAt    *time.Time `json:"last_run_at"`
}

// summary 总览页汇总
func (h *DashboardHandler) summary(ctx context.Context, c *app.RequestContext) {
	resp := dashboardSummary{
		DailyScrapes: []dashboardDailyTrend{},
		TaskSummary:  []dashboardTaskStatus{},
	}

	// 媒体统计（仅统计已选媒体库）
	if h.service != nil && h.service.IsAuthenticated() {
		stat, err := h.service.GetStatistics()
		if err == nil && stat != nil {
			resp.TotalMedia = stat.Total
		}
		// 本周新增媒体数量：遍历同步媒体库，查询近7天入库数量
		weekAgo := time.Now().AddDate(0, 0, -7).Unix()
		libIDs := h.service.TargetLibraryIDs()
		for _, libID := range libIDs {
			items, _, err := h.service.Client().ItemList(libID, []trimmedia.Type{trimmedia.TypeMovie, trimmedia.TypeTV, trimmedia.TypeVideo}, true, 1, 1, "create_time", "DESC")
			if err != nil {
				continue
			}
			for _, item := range items {
				if item.CreateTime >= weekAgo {
					resp.WeeklyNewMedia++
				}
			}
		}
	}

	// 累计刮削数量
	h.db.Model(&model.ScrapeLog{}).Count(&resp.TotalScrapes)

	// 本周刮削数量
	weekAgo := time.Now().AddDate(0, 0, -7)
	h.db.Model(&model.ScrapeLog{}).Where("created_at >= ?", weekAgo).Count(&resp.WeeklyScrapes)

	// 近15天每天刮削数量
	now := time.Now()
	for i := 14; i >= 0; i-- {
		dayStart := time.Date(now.Year(), now.Month(), now.Day()-i, 0, 0, 0, 0, now.Location())
		dayEnd := dayStart.AddDate(0, 0, 1)
		var count int64
		h.db.Model(&model.ScrapeLog{}).Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).Count(&count)
		resp.DailyScrapes = append(resp.DailyScrapes, dashboardDailyTrend{
			Date:  dayStart.Format("01-02"),
			Count: int(count),
		})
	}

	// 刮削任务状态
	var tasks []model.ScrapeTask
	h.db.Order("id asc").Find(&tasks)
	for _, task := range tasks {
		var runningCount int64
		h.db.Model(&model.TaskRunRecord{}).Where("task_id = ? AND status = ?", task.ID, "running").Count(&runningCount)
		resp.TaskSummary = append(resp.TaskSummary, dashboardTaskStatus{
			ID:          task.ID,
			Name:        task.Name,
			LibraryName: task.LibraryName,
			Enabled:     task.Enabled,
			IsRunning:   runningCount > 0,
			Interval:    task.Interval,
			LastRunAt:   task.LastRunAt,
		})
	}

	c.JSON(200, resp)
}
