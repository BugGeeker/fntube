package scheduler

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"fntube/internal/metatube"
	"fntube/internal/model"
	"fntube/internal/trimmedia"
)

// Scheduler 刮削计划任务调度器
type Scheduler struct {
	db      *gorm.DB
	service *trimmedia.Service
	ticker  *time.Ticker
	stopCh  chan struct{}
	running bool
	mu      sync.Mutex
}

// NewScheduler 创建调度器
func NewScheduler(db *gorm.DB, service *trimmedia.Service) *Scheduler {
	return &Scheduler{
		db:      db,
		service: service,
		stopCh:  make(chan struct{}),
	}
}

// Start 启动调度器
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return
	}
	s.running = true
	s.ticker = time.NewTicker(1 * time.Minute)
	go s.loop()
	log.Printf("[scheduler] 刮削计划任务调度器已启动")
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.running = false
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.stopCh)
	log.Printf("[scheduler] 刮削计划任务调度器已停止")
}

// loop 主循环，每分钟检查一次到期任务
func (s *Scheduler) loop() {
	for {
		select {
		case <-s.stopCh:
			return
		case <-s.ticker.C:
			s.checkAndRun()
		}
	}
}

// checkAndRun 检查并执行到期的任务
func (s *Scheduler) checkAndRun() {
	var tasks []model.ScrapeTask
	if err := s.db.Where("enabled = ?", true).Find(&tasks).Error; err != nil {
		return
	}

	now := time.Now()
	for _, task := range tasks {
		if task.LastRunAt != nil {
			nextRun := task.LastRunAt.Add(time.Duration(task.Interval) * time.Minute)
			if now.Before(nextRun) {
				continue
			}
		}
		// 执行任务
		go s.runTask(task.ID)
	}
}

// RunTask 立即执行指定任务（供 handler 调用）
func (s *Scheduler) RunTask(taskID uint) error {
	return s.runTask(taskID)
}

// ScrapeSingle 对单个媒体项执行刮削并记录日志（手动触发），返回番号和错误
func (s *Scheduler) ScrapeSingle(itemGUID string) (string, error) {
	if s.service == nil || !s.service.IsAuthenticated() {
		return "", fmt.Errorf("飞牛影视未连接")
	}

	// 获取媒体项信息
	item, err := s.service.GetItemInfo(itemGUID)
	if err != nil || item == nil {
		return "", fmt.Errorf("获取媒体项失败: %w", err)
	}

	// 读取 MetaTube 配置
	var cfg struct {
		Host            string
		Token            string
		TranslateMode   string
		TranslateEngine  string
		EngineConfig     string
	}
	if err := s.db.Table("meta_tube_configs").Order("id desc").First(&cfg).Error; err != nil {
		return "", fmt.Errorf("未配置 MetaTube")
	}
	if cfg.Host == "" {
		return "", fmt.Errorf("MetaTube 服务地址为空")
	}

	mtClient := metatube.NewClient(cfg.Host, cfg.Token)
	defer mtClient.Close()

	// 获取飞牛类型列表
	genres, _ := s.service.GetGenreList("zh-CN")
	genreMap := map[string]int{}
	for _, g := range genres {
		genreMap[g.Value] = g.ID
	}

	// 执行刮削
	number, err := s.scrapeItem(*item, mtClient, cfg.TranslateMode, cfg.TranslateEngine, cfg.EngineConfig, genreMap)
	if err != nil {
		return number, err
	}

	// 记录刮削日志（手动触发）
	s.db.Create(&model.ScrapeLog{
		ItemGUID: item.Guid,
		Title:    item.Title,
		Number:   number,
		Method:   model.ScrapeMethodManual,
	})

	return number, nil
}

// runTask 执行单个刮削任务
func (s *Scheduler) runTask(taskID uint) error {
	// 获取任务
	var task model.ScrapeTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return fmt.Errorf("任务不存在: %w", err)
	}

	// 更新执行时间
	now := time.Now()
	s.db.Model(&task).Update("last_run_at", now)

	log.Printf("[scheduler] 开始执行刮削任务: %s (媒体库: %s)", task.Name, task.LibraryName)

	// 检查飞牛服务是否可用
	if s.service == nil || !s.service.IsAuthenticated() {
		return fmt.Errorf("飞牛影视未连接")
	}

	// 获取媒体库所有条目
	items, _, err := s.service.GetItems(task.LibraryID, 0, -1)
	if err != nil {
		return fmt.Errorf("获取媒体列表失败: %w", err)
	}

	// 获取已刮削过的 item_guid 集合
	scrapedSet, err := s.getScrapedItems()
	if err != nil {
		log.Printf("[scheduler] 获取已刮削记录失败: %v", err)
	}

	// 读取 MetaTube 配置
	var cfg struct {
		Host           string
		Token          string
		TranslateMode  string
		TranslateEngine string
		EngineConfig   string
	}
	if err := s.db.Table("meta_tube_configs").Order("id desc").First(&cfg).Error; err != nil {
		return fmt.Errorf("未配置 MetaTube")
	}
	if cfg.Host == "" {
		return fmt.Errorf("MetaTube 服务地址为空")
	}

	mtClient := metatube.NewClient(cfg.Host, cfg.Token)
	defer mtClient.Close()

	// 获取飞牛类型列表（用于匹配 genres）
	genres, _ := s.service.GetGenreList("zh-CN")
	genreMap := map[string]int{}
	for _, g := range genres {
		genreMap[g.Value] = g.ID
	}

	scrapedCount := 0
	for _, item := range items {
		// 跳过已刮削过的
		if scrapedSet[item.Guid] {
			continue
		}

		log.Printf("[scheduler] 刮削: %s (%s)", item.Title, item.Guid)

		number, err := s.scrapeItem(item, mtClient, cfg.TranslateMode, cfg.TranslateEngine, cfg.EngineConfig, genreMap)
		if err != nil {
			log.Printf("[scheduler] 刮削失败 %s: %v", item.Title, err)
			continue
		}

		// 记录刮削日志
		s.db.Create(&model.ScrapeLog{
			ItemGUID: item.Guid,
			Title:    item.Title,
			Number:   number,
			Method:   model.ScrapeMethodAuto,
		})

		scrapedCount++
	}

	log.Printf("[scheduler] 刮削任务完成: %s, 共刮削 %d 项", task.Name, scrapedCount)
	return nil
}

// getScrapedItems 获取已刮削过的 item_guid 集合
func (s *Scheduler) getScrapedItems() (map[string]bool, error) {
	var logs []model.ScrapeLog
	if err := s.db.Select("DISTINCT item_guid").Find(&logs).Error; err != nil {
		return nil, err
	}
	result := map[string]bool{}
	for _, l := range logs {
		result[l.ItemGUID] = true
	}
	return result, nil
}

// scrapeItem 对单个媒体项执行刮削，返回番号
func (s *Scheduler) scrapeItem(item trimmedia.MediaServerItem, mtClient *metatube.Client, translateMode, translateEngine, engineConfig string, genreMap map[string]int) (string, error) {
	keyword := item.Title
	if keyword == "" {
		return "", fmt.Errorf("标题为空")
	}

	// 1. 搜索 MetaTube
	results, err := mtClient.SearchMovies(keyword)
	if err != nil {
		return "", fmt.Errorf("搜索失败: %w", err)
	}
	if len(results) == 0 {
		return "", fmt.Errorf("无搜索结果")
	}
	first := results[0]

	// 2. 获取影片详情
	info, err := mtClient.GetMovieInfo(first.Provider, first.ID)
	if err != nil || info == nil {
		return "", fmt.Errorf("获取详情失败: %w", err)
	}

	// 3. 获取编辑信息
	detail, err := s.service.GetEditDetail(item.Guid)
	if err != nil || detail == nil {
		return info.Number, fmt.Errorf("获取编辑信息失败: %w", err)
	}

	// 4. 填入编辑信息（仅覆盖未锁定字段）
	fillEditDetail(detail, info, genreMap)

	// 5. 下载并上传图片
	if !detail.PostersLocked && (info.CoverURL != "" || info.BigCoverURL != "" || info.ThumbURL != "") {
		posterURL := info.BigCoverURL
		if posterURL == "" {
			posterURL = info.CoverURL
		}
		if posterURL == "" {
			posterURL = info.ThumbURL
		}
		if path, err := s.downloadAndUploadImage(posterURL, "poster"); err == nil {
			detail.Posters = path
		}
	}
	if !detail.BackdropsLocked && (info.ThumbURL != "" || info.BigThumbURL != "") {
		backdropURL := info.BigThumbURL
		if backdropURL == "" {
			backdropURL = info.ThumbURL
		}
		if path, err := s.downloadAndUploadImage(backdropURL, "backdrop"); err == nil {
			detail.Backdrops = path
		}
	}

	// 6. 处理演职员
	if !detail.CreditsLocked && len(info.Actors) > 0 {
		s.fillCredits(detail, info.Actors)
	}

	// 7. 翻译
	s.applyTranslation(detail, info, mtClient, translateMode, translateEngine, engineConfig)

	// 8. 保存
	_, err = s.service.SaveEditDetail(detail)
	if err != nil {
		return info.Number, fmt.Errorf("保存失败: %w", err)
	}

	return info.Number, nil
}

// fillEditDetail 将 MetaTube 影片信息填入编辑详情
func fillEditDetail(detail *trimmedia.EditDetail, info *metatube.MovieInfo, genreMap map[string]int) {
	if !detail.TitleLocked && info.Number != "" {
		detail.Title = info.Number
		if info.Title != "" {
			detail.Title += " " + info.Title
		}
	}
	if !detail.OverviewLocked && info.Summary != "" {
		detail.Overview = info.Summary
	}
	if !detail.RatingLocked && info.Score > 0 {
		detail.Rating = info.Score
	}
	if !detail.ContentRatingLocked {
		detail.ContentRating = "JP-18+"
	}
	if !detail.AirDateLocked && info.ReleaseDate != "" {
		detail.AirDate = formatDate(info.ReleaseDate)
	}
	if !detail.GenresLocked && len(info.Genres) > 0 {
		var matched []int
		for _, g := range info.Genres {
			if id, ok := genreMap[g]; ok {
				matched = append(matched, id)
			}
		}
		if len(matched) > 0 {
			detail.Genres = matched
		}
	}
}

// fillCredits 填入演职员信息
func (s *Scheduler) fillCredits(detail *trimmedia.EditDetail, actors []string) {
	credits := make([]trimmedia.EditCredit, 0, len(actors))
	for idx, name := range actors {
		credit := trimmedia.EditCredit{
			Name:  name,
			Job:   "Actor",
			Role:  "",
			Order: idx + 1,
		}

		// 搜索飞牛已有演员
		persons, err := s.service.SearchPersons(name, 1, 10)
		if err == nil && len(persons) > 0 {
			credit.PersonGUID = persons[0].GUID
			credit.Name = persons[0].Name
			credit.ProfilePath = persons[0].Profile
		} else {
			// 尝试通过 MetaTube 搜索演员 → 下载图片 → 上传 → 创建
			guid, profilePath, err := s.importPerson(name)
			if err == nil {
				credit.PersonGUID = guid
				credit.ProfilePath = profilePath
			}
		}
		credits = append(credits, credit)
	}
	detail.Credits = credits
}

// importPerson 通过 MetaTube 搜索演员并导入飞牛
func (s *Scheduler) importPerson(name string) (string, string, error) {
	// 读取 MetaTube 配置
	var cfg struct {
		Host  string
		Token string
	}
	if err := s.db.Table("meta_tube_configs").Order("id desc").First(&cfg).Error; err != nil {
		return "", "", err
	}
	if cfg.Host == "" {
		return "", "", fmt.Errorf("MetaTube 服务地址为空")
	}

	mtClient := metatube.NewClient(cfg.Host, cfg.Token)
	defer mtClient.Close()

	actors, err := mtClient.SearchActors(name)
	if err != nil || len(actors) == 0 {
		return "", "", fmt.Errorf("未找到演员")
	}

	actor := actors[0]
	var imageURL string
	if len(actor.Images) > 0 {
		imageURL = actor.Images[0]
	}

	var profilePath string
	if imageURL != "" {
		imgResp, err := http.Get(imageURL)
		if err == nil && imgResp.StatusCode == http.StatusOK {
			imgData, _ := io.ReadAll(imgResp.Body)
			imgResp.Body.Close()
			filename := "actor_profile.jpg"
			if idx := strings.LastIndex(imageURL, "/"); idx >= 0 {
				filename = imageURL[idx+1:]
				if filename == "" || !strings.Contains(filename, ".") {
					filename = "actor_profile.jpg"
				}
			}
			profilePath, _ = s.service.UploadImage(imgData, filename, "poster")
		}
	}

	guid, err := s.service.CreatePerson(name, profilePath)
	if err != nil {
		return "", "", err
	}
	return guid, profilePath, nil
}

// applyTranslation 根据翻译配置翻译标题和简介
func (s *Scheduler) applyTranslation(detail *trimmedia.EditDetail, info *metatube.MovieInfo, mtClient *metatube.Client, mode, engine, engineConfig string) {
	if mode == "" || mode == "none" {
		return
	}

	shouldTranslateTitle := mode == "title" || mode == "title_and_summary"
	shouldTranslateOverview := mode == "summary" || mode == "title_and_summary"

	if shouldTranslateTitle && !detail.TitleLocked && info.Title != "" {
		if result, err := mtClient.Translate(info.Title, "", "zh-CN", engine, engineConfig); err == nil && result != nil {
			detail.Title = info.Number + " " + result.TranslatedText
		}
	}
	if shouldTranslateOverview && !detail.OverviewLocked && info.Summary != "" {
		if result, err := mtClient.Translate(info.Summary, "", "zh-CN", engine, engineConfig); err == nil && result != nil {
			detail.Overview = result.TranslatedText
		}
	}
}

// downloadAndUploadImage 下载网络图片并上传到飞牛
func (s *Scheduler) downloadAndUploadImage(url, imageType string) (string, error) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return "", fmt.Errorf("仅支持 http/https")
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载图片状态码: %d", resp.StatusCode)
	}

	imgData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	filename := "image.jpg"
	if idx := strings.LastIndex(url, "/"); idx >= 0 {
		name := url[idx+1:]
		if qIdx := strings.Index(name, "?"); qIdx >= 0 {
			name = name[:qIdx]
		}
		if name != "" && strings.Contains(name, ".") {
			filename = name
		}
	}

	return s.service.UploadImage(imgData, filename, imageType)
}

// formatDate 格式化日期为 YYYY-MM-DD
func formatDate(date string) string {
	if date == "" {
		return ""
	}
	// 尝试解析 ISO 日期
	for _, layout := range []string{time.RFC3339, "2006-01-02", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, date); err == nil {
			return t.Format("2006-01-02")
		}
	}
	// 截取前10位
	if len(date) >= 10 {
		return date[:10]
	}
	return date
}

// RunTaskWithContext 带上下文执行任务（用于 API 调用）
func (s *Scheduler) RunTaskWithContext(ctx context.Context, taskID uint) error {
	return s.runTask(taskID)
}
