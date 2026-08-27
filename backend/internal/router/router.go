package router

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"gorm.io/gorm"

	"fntube/internal/handler"
	"fntube/internal/scheduler"
	"fntube/internal/trimmedia"
)

// Register 注册所有路由
func Register(h *server.Hertz, db *gorm.DB, trimSvc *trimmedia.Service, sched *scheduler.Scheduler) {
	// 请求日志中间件
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()
		path := string(c.URI().Path())
		method := string(c.Method())

		c.Next(ctx)

		latency := time.Since(start)
		status := c.Response.StatusCode()
		hlog.CtxInfof(ctx, "[%s] %s %d %s", method, path, status, latency)
	})

	// 健康检查
	h.GET("/api/health", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	// 应用版本
	h.GET("/api/version", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(200, map[string]string{"version": os.Getenv("TRIM_APPVER")})
	})

	// 飞牛影视
	handler.RegisterTrimMediaHandlers(h, db, trimSvc)

	// MetaTube
	handler.RegisterMetaTubeHandlers(h, db)

	// 刮削日志
	handler.RegisterScrapeLogHandlers(h, db, sched)

	// 刮削计划任务
	handler.RegisterScrapeTaskHandlers(h, db, trimSvc, sched)

	// 刮削任务运行记录
	handler.RegisterTaskRunRecordHandlers(h, db)

	// 总览页
	handler.RegisterDashboardHandlers(h, db, trimSvc)

	// 静态文件托管（前端构建产物），优先从应用安装目录读取
	staticDir := "./app/www"
	if appDest := os.Getenv("TRIM_APPDEST"); appDest != "" {
		staticDir = filepath.Join(appDest, "app", "www")
	}
	h.Static("/", staticDir)
}
