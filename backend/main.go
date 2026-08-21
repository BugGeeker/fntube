package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"fntube/internal/database"
	"fntube/internal/model"
	"fntube/internal/router"
	"fntube/internal/trimmedia"

	"github.com/cloudwego/hertz/pkg/app/server"
	"gorm.io/gorm"
)

// initTrimMediaService 从数据库加载飞牛影视配置并初始化服务
func initTrimMediaService(db *gorm.DB) *trimmedia.Service {
	var cfg model.TrimMediaConfig
	if err := db.Order("id desc").First(&cfg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return trimmedia.NewService("", "", "", "", "", nil)
		}
		log.Printf("加载飞牛影视配置失败: %v", err)
		return trimmedia.NewService("", "", "", "", "", nil)
	}

	syncLibs := trimmedia.LoadSyncLibraries(cfg.SyncLibraries)
	svc := trimmedia.NewService(cfg.Host, cfg.Username, cfg.Password, cfg.AccessCode, cfg.PlayHost, syncLibs)

	// 尝试连接，失败不阻断启动
	if svc.IsConfigured() {
		if !svc.Reconnect() {
			log.Printf("飞牛影视连接失败，请检查服务端地址 %s", cfg.Host)
		}
	}
	return svc
}

func main() {
	// 数据库路径：优先使用飞牛环境变量，开发环境使用默认路径
	dbPath := os.Getenv("TRIM_PKGVAR")
	if dbPath == "" {
		dbPath = filepath.Join(os.TempDir(), "fntube")
		_ = os.MkdirAll(dbPath, 0755)
	}
	dbPath = filepath.Join(dbPath, "fntube.db")

	// 初始化数据库
	db, err := database.Init(dbPath)
	if err != nil {
		log.Fatalf("failed to init database: %v", err)
	}

	// 初始化飞牛影视服务
	trimSvc := initTrimMediaService(db)

	// 端口：优先使用飞牛注入的 {port} 环境变量
	port := os.Getenv("FN_APP_PORT")
	if port == "" {
		port = "12786"
	}

	// 创建 Hertz 服务
	h := server.Default(server.WithHostPorts(fmt.Sprintf(":%s", port)))

	// 注册路由
	router.Register(h, db, trimSvc)

	// 启动服务
	h.Spin()
}
