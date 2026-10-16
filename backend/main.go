package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fntube/internal/database"
	"fntube/internal/model"
	"fntube/internal/router"
	"fntube/internal/scheduler"
	"fntube/internal/trimmedia"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/network/standard"
	"gorm.io/gorm"
)

const gatewayPrefix = "/app/FnTube"

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

func createServer() (*server.Hertz, func(), error) {
	appDest := os.Getenv("TRIM_APPDEST")
	if os.Getenv("FNTUBE_UNIFIED_GATEWAY") != "1" {
		port := os.Getenv("FN_APP_PORT")
		if port == "" {
			port = "12786"
		}
		return server.Default(server.WithHostPorts(fmt.Sprintf(":%s", port))), func() {}, nil
	}
	if appDest == "" {
		return nil, nil, fmt.Errorf("TRIM_APPDEST is required for the unified gateway")
	}

	backendListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, nil, fmt.Errorf("listen on internal HTTP address: %w", err)
	}

	socketPath := filepath.Join(appDest, "app.sock")
	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		_ = backendListener.Close()
		return nil, nil, fmt.Errorf("remove stale gateway socket: %w", err)
	}
	gatewayListener, err := net.Listen("unix", socketPath)
	if err != nil {
		_ = backendListener.Close()
		return nil, nil, fmt.Errorf("listen on gateway socket %s: %w", socketPath, err)
	}
	// The fnOS gateway runs outside the app user/group and must be able to
	// connect to this socket after authenticating the request.
	if err := os.Chmod(socketPath, 0666); err != nil {
		_ = gatewayListener.Close()
		_ = backendListener.Close()
		_ = os.Remove(socketPath)
		return nil, nil, fmt.Errorf("set gateway socket permissions: %w", err)
	}

	target := &url.URL{Scheme: "http", Host: backendListener.Addr().String()}
	proxy := httputil.NewSingleHostReverseProxy(target)
	defaultDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		// Different fnOS gateway versions may preserve or strip gatewayPrefix.
		// Hertz always uses the prefixed routes, so normalize both forms here.
		if req.URL.Path != gatewayPrefix && !strings.HasPrefix(req.URL.Path, gatewayPrefix+"/") {
			if req.URL.Path == "/" {
				req.URL.Path = gatewayPrefix + "/"
			} else {
				req.URL.Path = gatewayPrefix + "/" + strings.TrimPrefix(req.URL.Path, "/")
			}
		}
		defaultDirector(req)
	}
	// A negative interval flushes every write, which keeps SSE events real-time.
	proxy.FlushInterval = -1
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("gateway proxy failed for %s %s: %v", r.Method, r.URL.Path, err)
		http.Error(w, "FnTube backend unavailable", http.StatusBadGateway)
	}
	gatewayServer := &http.Server{
		Handler:           proxy,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if err := gatewayServer.Serve(gatewayListener); err != nil && err != http.ErrServerClosed {
			log.Printf("gateway socket server stopped: %v", err)
		}
	}()

	cleanup := func() {
		_ = gatewayServer.Close()
		_ = gatewayListener.Close()
		_ = backendListener.Close()
		_ = os.Remove(socketPath)
	}
	h := server.Default(
		server.WithListener(backendListener),
		server.WithTransport(standard.NewTransporter),
		server.WithBasePath(gatewayPrefix),
	)
	log.Printf("fnOS gateway socket ready: %s -> http://%s", socketPath, backendListener.Addr())
	return h, cleanup, nil
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

	// 初始化刮削计划任务调度器
	sched := scheduler.NewScheduler(db, trimSvc)
	sched.Start()

	// 飞牛部署使用统一网关的 Unix Socket，本地开发仍使用 TCP 端口。
	h, cleanup, err := createServer()
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}
	defer cleanup()

	// 注册路由
	router.Register(h, db, trimSvc, sched)

	// 启动服务
	h.Spin()
}
