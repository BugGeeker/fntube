package handler

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"gorm.io/gorm"

	"fntube/internal/metatube"
	"fntube/internal/trimmedia"
)

// TrimMediaHandler 飞牛影视处理器
type TrimMediaHandler struct {
	db      *gorm.DB
	service *trimmedia.Service
}

// NewTrimMediaHandler 创建飞牛影视处理器
func NewTrimMediaHandler(db *gorm.DB, service *trimmedia.Service) *TrimMediaHandler {
	return &TrimMediaHandler{db: db, service: service}
}

// RegisterTrimMediaHandlers 注册飞牛影视相关路由
func RegisterTrimMediaHandlers(h *server.Hertz, db *gorm.DB, service *trimmedia.Service) {
	hd := NewTrimMediaHandler(db, service)
	g := h.Group("/api/trimmedia")
	g.GET("/config", hd.getConfig)
	g.POST("/config", hd.saveConfig)
	g.POST("/test", hd.testConnection)
	g.GET("/libraries", hd.getLibraries)
	g.GET("/libraries/:libraryId/view-type", hd.getLibraryViewType)
	g.GET("/items/:libraryId", hd.getItems)
	g.GET("/item/:itemId", hd.getItem)
	g.GET("/seasons/:tvId", hd.getSeasons)
	g.GET("/episodes/:seasonId", hd.getEpisodes)
	g.GET("/persons/:itemId", hd.getPersons)
	g.POST("/persons/search", hd.searchPersons)
	g.POST("/persons/import", hd.importPerson)
	g.POST("/image/download-upload", hd.downloadAndUploadImage)
	g.GET("/edit/:itemId", hd.getEditDetail)
	g.POST("/edit/:itemId", hd.saveEditDetail)
	g.GET("/playurl/:itemId", hd.getPlayURL)
	g.GET("/resume", hd.getResume)
	g.GET("/latest", hd.getLatest)
	g.GET("/statistics", hd.getStatistics)
	g.GET("/genres", hd.getGenres)
	g.POST("/genres/batch", hd.batchCreateGenres)
	g.GET("/countries", hd.getCountries)
	g.POST("/refresh", hd.refresh)
	g.GET("/search", hd.search)
	g.GET("/stream/:itemId", hd.getStreamList)
	g.GET("/img", hd.proxyImage)
}

// getConfig 获取飞牛影视配置
func (h *TrimMediaHandler) getConfig(ctx context.Context, c *app.RequestContext) {
	// 飞牛配置只保存一条记录
	var cfg trimmediaConfigView
	_ = h.db.Table("trim_media_configs").Order("id desc").First(&cfg).Error
	c.JSON(200, cfg)
}

// trimmediaConfigView 配置视图
type trimmediaConfigView struct {
	ID            uint   `json:"id"`
	Host          string `json:"host"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	AccessCode    string `json:"access_code"`
	PlayHost      string `json:"play_host"`
	SyncLibraries string `json:"sync_libraries"`
}

// saveConfig 保存飞牛影视配置
func (h *TrimMediaHandler) saveConfig(ctx context.Context, c *app.RequestContext) {
	var req struct {
		Host          string `json:"host"`
		Username      string `json:"username"`
		Password      string `json:"password"`
		AccessCode    string `json:"access_code"`
		PlayHost      string `json:"play_host"`
		SyncLibraries string `json:"sync_libraries"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}

	// 使用原生 SQL upsert，避免在 handler 层引用 model.TrimMediaConfig
	// 先尝试读取已有配置
	var id int64
	row := h.db.Table("trim_media_configs").Select("id").Order("id desc").Limit(1).Row()
	_ = row.Scan(&id)

	values := map[string]interface{}{
		"host":           req.Host,
		"username":       req.Username,
		"password":       req.Password,
		"access_code":    req.AccessCode,
		"play_host":      req.PlayHost,
		"sync_libraries": req.SyncLibraries,
	}

	if id > 0 {
		if err := h.db.Table("trim_media_configs").Where("id = ?", id).Updates(values).Error; err != nil {
			c.JSON(500, map[string]string{"error": err.Error()})
			return
		}
	} else {
		if err := h.db.Table("trim_media_configs").Create(values).Error; err != nil {
			c.JSON(500, map[string]string{"error": err.Error()})
			return
		}
	}

	// 更新运行中的服务并立即重连
	if h.service != nil {
		if h.service.UpdateConfig(req.Host, req.Username, req.Password, req.AccessCode, req.PlayHost, req.SyncLibraries) {
			c.JSON(200, map[string]string{"status": "ok", "connected": "true"})
			return
		}
		// 连接失败不阻断配置保存，返回提示
		c.JSON(200, map[string]string{"status": "ok", "connected": "false", "message": "配置已保存，但连接飞牛影视失败，请检查配置"})
		return
	}

	c.JSON(200, map[string]string{"status": "ok"})
}

// testConnection 测试连接
func (h *TrimMediaHandler) testConnection(ctx context.Context, c *app.RequestContext) {
	var req struct {
		Host       string `json:"host"`
		Username   string `json:"username"`
		Password   string `json:"password"`
		AccessCode string `json:"access_code"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}

	svc := trimmedia.NewService(req.Host, req.Username, req.Password, req.AccessCode, "", nil)
	ok := svc.Reconnect()
	if !ok {
		c.JSON(400, map[string]string{"error": "连接失败"})
		return
	}
	defer svc.Disconnect()
	libs, _ := svc.GetLibraries(false)
	c.JSON(200, map[string]interface{}{
		"status":    "ok",
		"version":   svc.Client().Version(),
		"user":      svc.Client().Token() != "",
		"libraries": libs,
	})
}

// getLibraries 列出媒体库
func (h *TrimMediaHandler) getLibraries(ctx context.Context, c *app.RequestContext) {
	if h.service == nil || !h.service.IsAuthenticated() {
		c.JSON(401, map[string]string{"error": "not authenticated"})
		return
	}
	libs, err := h.service.GetLibraries(false)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, libs)
}

// getLibraryViewType 获取媒体库布局类型
func (h *TrimMediaHandler) getLibraryViewType(ctx context.Context, c *app.RequestContext) {
	if h.service == nil || !h.service.IsAuthenticated() {
		c.JSON(401, map[string]string{"error": "not authenticated"})
		return
	}
	libraryID := string(c.Param("libraryId"))
	viewType, err := h.service.GetLibraryViewType(libraryID)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, map[string]string{"view_type": viewType})
}

// getItems 列出媒体库下的项目
func (h *TrimMediaHandler) getItems(ctx context.Context, c *app.RequestContext) {
	if h.service == nil || !h.service.IsAuthenticated() {
		c.JSON(401, map[string]string{"error": "not authenticated"})
		return
	}
	libID := string(c.Param("libraryId"))
	start, _ := strconv.Atoi(string(c.Query("start")))
	if start < 0 {
		start = 0
	}
	limit, _ := strconv.Atoi(string(c.Query("limit")))
	if limit <= 0 {
		limit = 20
	}
	items, total, err := h.service.GetItems(libID, start, limit)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, map[string]interface{}{
		"total": total,
		"items": items,
	})
}

// getItem 媒体详情
func (h *TrimMediaHandler) getItem(ctx context.Context, c *app.RequestContext) {
	if h.service == nil || !h.service.IsAuthenticated() {
		c.JSON(401, map[string]string{"error": "not authenticated"})
		return
	}
	itemID := string(c.Param("itemId"))
	item, err := h.service.GetItemInfo(itemID)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, item)
}

// getSeasons 季列表
func (h *TrimMediaHandler) getSeasons(ctx context.Context, c *app.RequestContext) {
	if h.service == nil || !h.service.IsAuthenticated() {
		c.JSON(401, map[string]string{"error": "not authenticated"})
		return
	}
	tvID := string(c.Param("tvId"))
	if h.service.Client() == nil {
		c.JSON(500, map[string]string{"error": "no client"})
		return
	}
	seasons, err := h.service.Client().SeasonList(tvID)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, seasons)
}

// getEpisodes 集列表
func (h *TrimMediaHandler) getEpisodes(ctx context.Context, c *app.RequestContext) {
	if h.service == nil || !h.service.IsAuthenticated() {
		c.JSON(401, map[string]string{"error": "not authenticated"})
		return
	}
	seasonID := string(c.Param("seasonId"))
	if h.service.Client() == nil {
		c.JSON(500, map[string]string{"error": "no client"})
		return
	}
	episodes, err := h.service.Client().EpisodeList(seasonID)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, episodes)
}

// getPersons 媒体演职员列表
func (h *TrimMediaHandler) getPersons(ctx context.Context, c *app.RequestContext) {
	if h.service == nil || !h.service.IsAuthenticated() {
		c.JSON(401, map[string]string{"error": "not authenticated"})
		return
	}
	itemID := string(c.Param("itemId"))
	persons, err := h.service.GetPersonList(itemID)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, persons)
}

// searchPersons 搜索演员
func (h *TrimMediaHandler) searchPersons(ctx context.Context, c *app.RequestContext) {
	if h.service == nil || !h.service.IsAuthenticated() {
		c.JSON(401, map[string]string{"error": "not authenticated"})
		return
	}
	var req struct {
		Keyword  string `json:"keyword"`
		Page     int    `json:"page"`
		PageSize int    `json:"page_size"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}
	if req.Keyword == "" {
		c.JSON(400, map[string]string{"error": "keyword 不能为空"})
		return
	}
	results, err := h.service.SearchPersons(req.Keyword, req.Page, req.PageSize)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, results)
}

// importPerson 从 MetaTube 搜索演员，下载图片并上传飞牛后创建演员
// 完整流程：MetaTube搜索演员 → 下载图片 → 飞牛上传图片 → 飞牛创建演员
func (h *TrimMediaHandler) importPerson(ctx context.Context, c *app.RequestContext) {
	if h.service == nil || !h.service.IsAuthenticated() {
		c.JSON(401, map[string]string{"error": "not authenticated"})
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}
	if req.Name == "" {
		c.JSON(400, map[string]string{"error": "name 不能为空"})
		return
	}

	// 1. 从数据库读取 MetaTube 配置
	var cfg struct {
		Host  string
		Token string
	}
	if err := h.db.Table("meta_tube_configs").Order("id desc").First(&cfg).Error; err != nil {
		c.JSON(400, map[string]string{"error": "未配置 MetaTube"})
		return
	}
	if cfg.Host == "" {
		c.JSON(400, map[string]string{"error": "MetaTube 服务地址为空"})
		return
	}

	// 2. 通过 MetaTube 搜索演员
	mtClient := metatube.NewClient(cfg.Host, cfg.Token)
	defer mtClient.Close()

	actors, err := mtClient.SearchActors(req.Name)
	if err != nil {
		c.JSON(500, map[string]string{"error": "MetaTube 搜索演员失败: " + err.Error()})
		return
	}
	if len(actors) == 0 {
		c.JSON(404, map[string]string{"error": "MetaTube 未找到该演员"})
		return
	}

	// 取第一个搜索结果
	actor := actors[0]
	// 选取第一张图片作为头像
	var imageURL string
	if len(actor.Images) > 0 {
		imageURL = actor.Images[0]
	}

	var profilePath string

	// 3. 如果有图片，下载并上传到飞牛
	if imageURL != "" {
		// 下载图片
		imgResp, err := http.Get(imageURL)
		if err != nil {
			c.JSON(500, map[string]string{"error": "下载演员图片失败: " + err.Error()})
			return
		}
		defer imgResp.Body.Close()

		if imgResp.StatusCode != http.StatusOK {
			c.JSON(500, map[string]string{"error": fmt.Sprintf("下载图片状态码: %d", imgResp.StatusCode)})
			return
		}

		imgData, err := io.ReadAll(imgResp.Body)
		if err != nil {
			c.JSON(500, map[string]string{"error": "读取图片数据失败: " + err.Error()})
			return
		}

		// 从 URL 推断文件名和扩展名
		filename := "actor_profile.jpg"
		if idx := strings.LastIndex(imageURL, "/"); idx >= 0 {
			filename = imageURL[idx+1:]
			if filename == "" || !strings.Contains(filename, ".") {
				filename = "actor_profile.jpg"
			}
		}

		// 上传图片到飞牛
		profilePath, err = h.service.UploadImage(imgData, filename, "poster")
		if err != nil {
			c.JSON(500, map[string]string{"error": "上传图片到飞牛失败: " + err.Error()})
			return
		}
	}

	// 4. 在飞牛创建演员
	guid, err := h.service.CreatePerson(req.Name, profilePath)
	if err != nil {
		c.JSON(500, map[string]string{"error": "创建演员失败: " + err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{
		"guid":         guid,
		"name":         req.Name,
		"profile_path": profilePath,
	})
}

// downloadAndUploadImage 下载 http 网络图片并上传到飞牛，返回飞牛图片路径
// body: { url, type }  type 为图片类型（如 poster、backdrop），默认 poster
func (h *TrimMediaHandler) downloadAndUploadImage(ctx context.Context, c *app.RequestContext) {
	if h.service == nil || !h.service.IsAuthenticated() {
		c.JSON(401, map[string]string{"error": "not authenticated"})
		return
	}
	var req struct {
		URL  string `json:"url"`
		Type string `json:"type"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}
	if req.URL == "" {
		c.JSON(400, map[string]string{"error": "url 不能为空"})
		return
	}
	// 仅允许 http/https 网络图片，防止传入本地路径
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		c.JSON(400, map[string]string{"error": "仅支持 http/https 网络图片"})
		return
	}
	if req.Type == "" {
		req.Type = "poster"
	}

	// 下载图片
	imgResp, err := http.Get(req.URL)
	if err != nil {
		c.JSON(500, map[string]string{"error": "下载图片失败: " + err.Error()})
		return
	}
	defer imgResp.Body.Close()
	if imgResp.StatusCode != http.StatusOK {
		c.JSON(500, map[string]string{"error": fmt.Sprintf("下载图片状态码: %d", imgResp.StatusCode)})
		return
	}
	imgData, err := io.ReadAll(imgResp.Body)
	if err != nil {
		c.JSON(500, map[string]string{"error": "读取图片数据失败: " + err.Error()})
		return
	}

	// 从 URL 推断文件名与扩展名（去掉查询参数）
	filename := "image.jpg"
	if idx := strings.LastIndex(req.URL, "/"); idx >= 0 {
		name := req.URL[idx+1:]
		if qIdx := strings.Index(name, "?"); qIdx >= 0 {
			name = name[:qIdx]
		}
		if name != "" && strings.Contains(name, ".") {
			filename = name
		}
	}

	// 上传到飞牛临时存储，返回飞牛图片路径
	path, err := h.service.UploadImage(imgData, filename, req.Type)
	if err != nil {
		c.JSON(500, map[string]string{"error": "上传图片到飞牛失败: " + err.Error()})
		return
	}
	c.JSON(200, map[string]string{"path": path})
}

// getEditDetail 获取媒体项编辑信息
func (h *TrimMediaHandler) getEditDetail(ctx context.Context, c *app.RequestContext) {
	if h.service == nil || !h.service.IsAuthenticated() {
		c.JSON(401, map[string]string{"error": "not authenticated"})
		return
	}
	itemID := string(c.Param("itemId"))
	detail, err := h.service.GetEditDetail(itemID)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, detail)
}

// saveEditDetail 保存媒体项编辑信息
func (h *TrimMediaHandler) saveEditDetail(ctx context.Context, c *app.RequestContext) {
	if h.service == nil || !h.service.IsAuthenticated() {
		c.JSON(401, map[string]string{"error": "not authenticated"})
		return
	}
	var detail trimmedia.EditDetail
	if err := c.BindJSON(&detail); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}
	// 路径参数中的 itemId 为权威值，覆盖 body 内的 item_guid
	if itemID := string(c.Param("itemId")); itemID != "" {
		detail.ItemGUID = itemID
	}
	ok, err := h.service.SaveEditDetail(&detail)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, map[string]bool{"success": ok})
}

// getPlayURL 获取播放链接
func (h *TrimMediaHandler) getPlayURL(ctx context.Context, c *app.RequestContext) {
	if h.service == nil || !h.service.IsAuthenticated() {
		c.JSON(401, map[string]string{"error": "not authenticated"})
		return
	}
	itemID := string(c.Param("itemId"))
	url, err := h.service.GetPlayURL(itemID)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, map[string]string{"url": url})
}

// getResume 继续观看
func (h *TrimMediaHandler) getResume(ctx context.Context, c *app.RequestContext) {
	if h.service == nil || !h.service.IsAuthenticated() {
		c.JSON(401, map[string]string{"error": "not authenticated"})
		return
	}
	num := 12
	if n := string(c.Query("num")); n != "" {
		if v, err := strconv.Atoi(n); err == nil && v > 0 {
			num = v
		}
	}
	items, err := h.service.GetResume(num)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, items)
}

// getLatest 最近更新
func (h *TrimMediaHandler) getLatest(ctx context.Context, c *app.RequestContext) {
	if h.service == nil || !h.service.IsAuthenticated() {
		c.JSON(401, map[string]string{"error": "not authenticated"})
		return
	}
	num := 20
	if n := string(c.Query("num")); n != "" {
		if v, err := strconv.Atoi(n); err == nil && v > 0 {
			num = v
		}
	}
	items, err := h.service.GetLatest(num)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, items)
}

// getStatistics 媒体统计
func (h *TrimMediaHandler) getStatistics(ctx context.Context, c *app.RequestContext) {
	if h.service == nil || !h.service.IsAuthenticated() {
		c.JSON(401, map[string]string{"error": "not authenticated"})
		return
	}
	stat, err := h.service.GetStatistics()
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, stat)
}

// getGenres 媒体类型列表
func (h *TrimMediaHandler) getGenres(ctx context.Context, c *app.RequestContext) {
	if h.service == nil || !h.service.IsAuthenticated() {
		c.JSON(401, map[string]string{"error": "not authenticated"})
		return
	}
	lan := string(c.Query("lan"))
	genres, err := h.service.GetGenreList(lan)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, genres)
}

// getCountries 国家地区列表
func (h *TrimMediaHandler) getCountries(ctx context.Context, c *app.RequestContext) {
	if h.service == nil || !h.service.IsAuthenticated() {
		c.JSON(401, map[string]string{"error": "not authenticated"})
		return
	}
	lan := string(c.Query("lan"))
	countries, err := h.service.GetCountryList(lan)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, countries)
}

// batchCreateGenres 批量新增自定义分类
func (h *TrimMediaHandler) batchCreateGenres(ctx context.Context, c *app.RequestContext) {
	if h.service == nil || !h.service.IsAuthenticated() {
		c.JSON(401, map[string]string{"error": "not authenticated"})
		return
	}
	var req struct {
		Values []string `json:"values"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": err.Error()})
		return
	}
	if len(req.Values) == 0 {
		c.JSON(400, map[string]string{"error": "values 不能为空"})
		return
	}
	genres, err := h.service.BatchCreateGenres(req.Values)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, genres)
}

// refresh 刷新媒体库
func (h *TrimMediaHandler) refresh(ctx context.Context, c *app.RequestContext) {
	if h.service == nil || !h.service.IsAuthenticated() {
		c.JSON(401, map[string]string{"error": "not authenticated"})
		return
	}
	var body struct {
		Paths []string `json:"paths"`
	}
	_ = c.BindJSON(&body)
	if len(body.Paths) == 0 {
		ok, err := h.service.RefreshRootLibrary()
		if err != nil {
			c.JSON(500, map[string]string{"error": err.Error()})
			return
		}
		c.JSON(200, map[string]bool{"success": ok})
		return
	}
	ok, err := h.service.RefreshLibrary(body.Paths)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, map[string]bool{"success": ok})
}

// search 搜索
func (h *TrimMediaHandler) search(ctx context.Context, c *app.RequestContext) {
	if h.service == nil || !h.service.IsAuthenticated() {
		c.JSON(401, map[string]string{"error": "not authenticated"})
		return
	}
	q := string(c.Query("q"))
	if q == "" {
		c.JSON(400, map[string]string{"error": "q is required"})
		return
	}
	items, err := h.service.Search(q)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, items)
}

// getStreamList 获取媒体文件/视频/音频/字幕流信息
func (h *TrimMediaHandler) getStreamList(ctx context.Context, c *app.RequestContext) {
	if h.service == nil || !h.service.IsAuthenticated() {
		c.JSON(401, map[string]string{"error": "not authenticated"})
		return
	}
	itemID := string(c.Param("itemId"))
	result, err := h.service.GetStreamList(itemID)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(200, result)
}

// proxyImage 代理飞牛图片，携带登录会话 Cookie 访问
// query: path = 飞牛相对图片路径（/api/v1/sys/img/...，可含 ?w= 尺寸参数）
func (h *TrimMediaHandler) proxyImage(ctx context.Context, c *app.RequestContext) {
	if h.service == nil || !h.service.IsAuthenticated() {
		c.AbortWithStatus(401)
		return
	}
	cli := h.service.Client()
	if cli == nil {
		c.AbortWithStatus(500)
		return
	}
	path := string(c.Query("path"))
	if path == "" {
		c.AbortWithStatus(400)
		return
	}
	// 仅允许代理飞牛图片路径，防止 SSRF
	if !strings.HasPrefix(path, "/api/v1/sys/img/") {
		c.AbortWithStatus(403)
		return
	}

	data, contentType, err := cli.ProxyImage(path)
	if err != nil {
		c.AbortWithStatus(502)
		return
	}

	etag := fmt.Sprintf(`"%x"`, md5.Sum(data))
	c.Header("ETag", etag)
	c.Header("Cache-Control", "public, max-age=86400")
	if ifNoneMatch := string(c.GetHeader("If-None-Match")); ifNoneMatch != "" && ifNoneMatch == etag {
		c.SetStatusCode(304)
		return
	}
	c.Data(200, contentType, data)
}
