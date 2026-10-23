package router

import (
	"bytes"
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

	// 影片搜索
	handler.RegisterMovieSearchHandlers(h, db)

	// 光鸭云盘
	handler.RegisterGuangyaHandlers(h, db, trimSvc, sched)

	// SmartStrm
	handler.RegisterSmartStrmHandlers(h, db)

	// 存储配置
	handler.RegisterStorageConfigHandlers(h, db)

	// 刮削日志
	handler.RegisterScrapeLogHandlers(h, db, sched)

	// 刮削计划任务
	handler.RegisterScrapeTaskHandlers(h, db, trimSvc, sched)

	// 刮削任务运行记录
	handler.RegisterTaskRunRecordHandlers(h, db)

	// 总览页
	handler.RegisterDashboardHandlers(h, db, trimSvc)

	// 系统能力测试
	handler.RegisterSystemTestHandlers(h)

	// 静态文件托管（前端构建产物）。不同 fnOS 安装方式的目录层级可能不同，
	// 按实际存在的 index.html 选择目录。
	staticCandidates := []string{"./app/www"}
	if appDest := os.Getenv("TRIM_APPDEST"); appDest != "" {
		staticCandidates = append([]string{
			filepath.Join(appDest, "www"),
			filepath.Join(appDest, "app", "www"),
		}, staticCandidates...)
	}
	if executable, err := os.Executable(); err == nil {
		staticCandidates = append(staticCandidates,
			filepath.Join(filepath.Dir(executable), "..", "www"),
		)
	}

	staticDir := staticCandidates[0]
	for _, candidate := range staticCandidates {
		indexPath := filepath.Join(candidate, "index.html")
		if info, err := os.Stat(indexPath); err == nil && !info.IsDir() {
			staticDir = candidate
			break
		}
	}
	indexPath := filepath.Join(staticDir, "index.html")
	hlog.Infof("frontend static directory: %s (index exists: %t)", staticDir, fileExists(indexPath))
	serveIndex := func(ctx context.Context, c *app.RequestContext) {
		c.File(indexPath)
	}
	h.GET("/", serveIndex)
	h.GET("/index.html", serveIndex)
	h.StaticFS("/", &app.FS{
		Root:        staticDir,
		PathRewrite: stripGatewayPrefix,
	})
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func stripGatewayPrefix(c *app.RequestContext) []byte {
	path := c.Path()
	prefix := []byte("/app/FnTube")
	if bytes.Equal(path, prefix) {
		return []byte("/")
	}
	if bytes.HasPrefix(path, append(prefix, '/')) {
		return path[len(prefix):]
	}
	return path
}
