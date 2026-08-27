package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"fntube/badge"
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
		// 检查任务是否正在运行，防止重复执行
		var runningCount int64
		s.db.Model(&model.TaskRunRecord{}).Where("task_id = ? AND status = ?", task.ID, "running").Count(&runningCount)
		if runningCount > 0 {
			log.Printf("[scheduler] 任务 %s 正在运行，跳过本次调度", task.Name)
			continue
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
		Token           string
		TranslateMode   string
		TranslateEngine string
		EngineConfig    string
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
	number, err := s.scrapeItem(*item, mtClient, cfg.TranslateMode, cfg.TranslateEngine, cfg.EngineConfig, genreMap, model.ScrapeMethodManual, 0)
	if err != nil {
		return number, err
	}

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

	// 创建运行记录
	runRecord := model.TaskRunRecord{
		TaskID:      task.ID,
		TaskName:    task.Name,
		LibraryName: task.LibraryName,
		StartTime:   now,
		Status:      "running",
	}
	s.db.Create(&runRecord)

	// 记录运行结果到运行记录
	finishRecord := func(successCount, completedCount, failedCount int64, status, errMsg string) {
		endTime := time.Now()
		s.db.Model(&runRecord).Updates(map[string]interface{}{
			"end_time":        endTime,
			"duration":        int64(endTime.Sub(now).Seconds()),
			"success_count":   successCount,
			"completed_count": completedCount,
			"failed_count":    failedCount,
			"status":          status,
			"error":           errMsg,
			"updated_at":      time.Now(),
		})
	}

	// 检查飞牛服务是否可用
	if s.service == nil || !s.service.IsAuthenticated() {
		finishRecord(0, 0, 0, "error", "飞牛影视未连接")
		return fmt.Errorf("飞牛影视未连接")
	}

	// 获取媒体库所有条目
	items, _, err := s.service.GetItems(task.LibraryID, 0, -1)
	if err != nil {
		finishRecord(0, 0, 0, "error", "获取媒体列表失败: "+err.Error())
		return fmt.Errorf("获取媒体列表失败: %w", err)
	}

	// 获取已刮削过的 item_guid 集合
	scrapedSet, err := s.getScrapedItems()
	if err != nil {
		log.Printf("[scheduler] 获取已刮削记录失败: %v", err)
	}

	// 读取 MetaTube 配置
	var cfg struct {
		Host            string
		Token           string
		TranslateMode   string
		TranslateEngine string
		EngineConfig    string
	}
	if err := s.db.Table("meta_tube_configs").Order("id desc").First(&cfg).Error; err != nil {
		finishRecord(0, 0, 0, "error", "未配置 MetaTube")
		return fmt.Errorf("未配置 MetaTube")
	}
	if cfg.Host == "" {
		finishRecord(0, 0, 0, "error", "MetaTube 服务地址为空")
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
	failedCount := int64(0)
	for _, item := range items {
		// 跳过已刮削过的
		if scrapedSet[item.Guid] {
			continue
		}

		log.Printf("[scheduler] 刮削: %s (%s)", item.Title, item.Guid)

		_, err := s.scrapeItem(item, mtClient, cfg.TranslateMode, cfg.TranslateEngine, cfg.EngineConfig, genreMap, model.ScrapeMethodAuto, runRecord.ID)
		if err != nil {
			log.Printf("[scheduler] 刮削失败 %s: %v", item.Title, err)
			failedCount++
			continue
		}

		scrapedCount++
	}

	// 统计本次运行产生的成功和完成数量（通过 task_run_id 关联查询）
	var successCount, completedCount int64
	s.db.Model(&model.ScrapeLog{}).Where("task_run_id = ? AND status = ?", runRecord.ID, model.ScrapeStatusSuccess).Count(&successCount)
	s.db.Model(&model.ScrapeLog{}).Where("task_run_id = ? AND status = ?", runRecord.ID, model.ScrapeStatusCompleted).Count(&completedCount)

	log.Printf("[scheduler] 刮削任务完成: %s, 成功 %d, 完成 %d, 失败 %d", task.Name, successCount, completedCount, failedCount)

	// 仅当刮削数量不为0时保留记录，否则删除
	totalScraped := successCount + completedCount + failedCount
	if totalScraped == 0 {
		s.db.Delete(&runRecord)
	} else {
		finishRecord(successCount, completedCount, failedCount, "done", "")
	}

	return nil
}

// getScrapedItems 获取已刮削完成/成功的 item_guid 集合（跳过失败和进行中的）
func (s *Scheduler) getScrapedItems() (map[string]bool, error) {
	var logs []model.ScrapeLog
	if err := s.db.Select("DISTINCT item_guid").Where("status IN ?", []string{model.ScrapeStatusCompleted, model.ScrapeStatusSuccess}).Find(&logs).Error; err != nil {
		return nil, err
	}
	result := map[string]bool{}
	for _, l := range logs {
		result[l.ItemGUID] = true
	}
	return result, nil
}

// scrapeItem 对单个媒体项执行刮削，返回番号
func (s *Scheduler) scrapeItem(item trimmedia.MediaServerItem, mtClient *metatube.Client, translateMode, translateEngine, engineConfig string, genreMap map[string]int, method string, taskRunID uint) (string, error) {
	streamList, err := s.service.GetStreamList(item.Guid)
	if err != nil {
		return "", fmt.Errorf("获取媒体流信息失败: %w", err)
	}
	if len(streamList.Files) == 0 || streamList.Files[0].FileName == "" {
		return "", fmt.Errorf("媒体文件名为空")
	}
	keyword := streamList.Files[0].FileName
	if extIdx := strings.LastIndex(keyword, "."); extIdx > 0 {
		keyword = keyword[:extIdx]
	}

	// 创建刮削日志记录（开始时记录）
	logEntry := model.ScrapeLog{
		ItemGUID:  item.Guid,
		Title:     item.Title,
		Method:    method,
		TaskRunID: taskRunID,
		Status:    model.ScrapeStatusInProgress,
		Steps:     "[]",
	}
	s.db.Create(&logEntry)

	// 辅助：记录步骤（同步骤名只保留一条，后调用覆盖前调用的状态）
	stepRecords := []map[string]string{}
	addStep := func(step, status, errMsg string) {
		// 查找已有步骤，找到则更新，找不到则追加
		found := false
		for i := range stepRecords {
			if stepRecords[i]["step"] == step {
				stepRecords[i]["status"] = status
				if errMsg != "" {
					stepRecords[i]["error"] = errMsg
				} else {
					delete(stepRecords[i], "error")
				}
				found = true
				break
			}
		}
		if !found {
			entry := map[string]string{"step": step, "status": status}
			if errMsg != "" {
				entry["error"] = errMsg
			}
			stepRecords = append(stepRecords, entry)
		}
		stepsJSON, _ := json.Marshal(stepRecords)
		s.db.Model(&logEntry).Updates(map[string]interface{}{
			"steps":      string(stepsJSON),
			"updated_at": time.Now(),
		})
	}
	// 辅助：更新状态
	updateStatus := func(status, errMsg string) {
		updates := map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}
		if errMsg != "" {
			updates["error"] = errMsg
		}
		s.db.Model(&logEntry).Updates(updates)
	}

	// 1. 搜索 MetaTube
	addStep(model.StepSearch, "running", "")
	results, err := mtClient.SearchMovies(keyword)
	if err != nil {
		addStep(model.StepSearch, "failed", err.Error())
		updateStatus(model.ScrapeStatusFailed, "搜索失败: "+err.Error())
		return "", fmt.Errorf("搜索失败: %w", err)
	}
	if len(results) == 0 {
		addStep(model.StepSearch, "failed", "未查询到影片")
		updateStatus(model.ScrapeStatusFailed, "未查询到影片")
		return "", fmt.Errorf("无搜索结果")
	}
	addStep(model.StepSearch, "success", "")
	first := results[0]

	// 2. 获取影片详情
	addStep(model.StepGetDetail, "running", "")
	info, err := mtClient.GetMovieInfo(first.Provider, first.ID)
	if err != nil || info == nil {
		errMsg := "获取详情失败"
		if err != nil {
			errMsg += ": " + err.Error()
		}
		addStep(model.StepGetDetail, "failed", errMsg)
		updateStatus(model.ScrapeStatusFailed, errMsg)
		return "", fmt.Errorf("获取详情失败: %w", err)
	}
	addStep(model.StepGetDetail, "success", "")

	// 更新日志中的番号
	s.db.Model(&logEntry).Updates(map[string]interface{}{
		"number":     info.Number,
		"updated_at": time.Now(),
	})

	// 3. 获取编辑信息
	detail, err := s.service.GetEditDetail(item.Guid)
	if err != nil || detail == nil {
		return info.Number, fmt.Errorf("获取编辑信息失败: %w", err)
	}

	// 4. 填入编辑信息（仅覆盖未锁定字段）
	fillEditDetail(detail, info, genreMap)

	// 5. 下载并上传图片（不阻塞后续流程）
	if !detail.PostersLocked && (info.CoverURL != "" || info.BigCoverURL != "" || info.ThumbURL != "") {
		posterURL := info.BigCoverURL
		if posterURL == "" {
			posterURL = info.CoverURL
		}
		if posterURL == "" {
			posterURL = info.ThumbURL
		}
		addStep(model.StepDownloadPoster, "running", "")
		if path, err := s.downloadAndUploadPoster(posterURL, keyword); err == nil {
			detail.Posters = path
			addStep(model.StepDownloadPoster, "success", "")
		} else {
			addStep(model.StepDownloadPoster, "failed", err.Error())
		}
	}
	if !detail.BackdropsLocked && (info.ThumbURL != "" || info.BigThumbURL != "") {
		backdropURL := info.BigThumbURL
		if backdropURL == "" {
			backdropURL = info.ThumbURL
		}
		addStep(model.StepDownloadBackdrop, "running", "")
		if path, err := s.downloadAndUploadImage(backdropURL, "backdrop"); err == nil {
			detail.Backdrops = path
			addStep(model.StepDownloadBackdrop, "success", "")
		} else {
			addStep(model.StepDownloadBackdrop, "failed", err.Error())
		}
	}

	// 6. 处理演职员（不阻塞后续流程）
	if !detail.CreditsLocked && len(info.Actors) > 0 {
		addStep(model.StepSearchActor, "running", "")
		if err := s.fillCredits(detail, info.Actors); err == nil {
			addStep(model.StepSearchActor, "success", "")
		} else {
			addStep(model.StepSearchActor, "failed", err.Error())
		}
	}

	// 7. 翻译（不阻塞后续流程）
	if translateMode != "" && translateMode != "none" {
		addStep(model.StepTranslate, "running", "")
		if err := s.applyTranslation(detail, info, mtClient, translateMode, translateEngine, engineConfig); err == nil {
			addStep(model.StepTranslate, "success", "")
		} else {
			addStep(model.StepTranslate, "failed", err.Error())
		}
	}

	// 8. 保存
	_, err = s.service.SaveEditDetail(detail)
	if err != nil {
		updateStatus(model.ScrapeStatusFailed, "保存失败: "+err.Error())
		return info.Number, fmt.Errorf("保存失败: %w", err)
	}

	// 保存成功 → 根据步骤结果标记状态：全部成功为"成功"，有非阻塞失败为"完成"
	hasFailedStep := false
	for _, step := range stepRecords {
		if step["status"] == "failed" {
			hasFailedStep = true
			break
		}
	}
	if hasFailedStep {
		updateStatus(model.ScrapeStatusCompleted, "")
	} else {
		updateStatus(model.ScrapeStatusSuccess, "")
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

// fillCredits 填入演职员信息，返回错误（不阻塞，仅记录）
func (s *Scheduler) fillCredits(detail *trimmedia.EditDetail, actors []string) error {
	credits := make([]trimmedia.EditCredit, 0, len(actors))
	failed := 0
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
			} else {
				failed++
			}
		}
		credits = append(credits, credit)
	}
	detail.Credits = credits
	if failed > 0 {
		return fmt.Errorf("%d 名演员导入失败", failed)
	}
	return nil
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

// applyTranslation 根据翻译配置翻译标题和简介，返回错误（不阻塞，仅记录）
func (s *Scheduler) applyTranslation(detail *trimmedia.EditDetail, info *metatube.MovieInfo, mtClient *metatube.Client, mode, engine, engineConfig string) error {
	if mode == "" || mode == "none" {
		return nil
	}

	shouldTranslateTitle := mode == "title" || mode == "title_and_summary"
	shouldTranslateOverview := mode == "summary" || mode == "title_and_summary"

	var errs []string

	if shouldTranslateTitle && !detail.TitleLocked && info.Title != "" {
		result, err := mtClient.Translate(info.Title, "", "zh-CN", engine, engineConfig)
		if err == nil && result != nil {
			detail.Title = info.Number + " " + result.TranslatedText
		} else if err != nil {
			errs = append(errs, "标题翻译失败: "+err.Error())
		}
	}
	if shouldTranslateOverview && !detail.OverviewLocked && info.Summary != "" {
		result, err := mtClient.Translate(info.Summary, "", "zh-CN", engine, engineConfig)
		if err == nil && result != nil {
			detail.Overview = result.TranslatedText
		} else if err != nil {
			errs = append(errs, "简介翻译失败: "+err.Error())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, "; "))
	}
	return nil
}

// downloadAndUploadPoster 下载封面、按关键词添加徽标并上传到飞牛。
func (s *Scheduler) downloadAndUploadPoster(url, keyword string) (string, error) {
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
	badgeName := posterBadgeName(keyword)
	if badgeName == "" {
		return s.service.UploadImage(imgData, imageFilename(url), "poster")
	}

	poster, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return "", fmt.Errorf("解析封面图片失败: %w", err)
	}
	badgeData, err := badge.Read(badgeName)
	if err != nil {
		return "", fmt.Errorf("读取徽标失败: %w", err)
	}
	badge, err := png.Decode(bytes.NewReader(badgeData))
	if err != nil {
		return "", fmt.Errorf("解析徽标失败: %w", err)
	}

	posterBounds := poster.Bounds()
	badge = scaleToFit(badge, posterBounds.Dx()/2, posterBounds.Dy()/2)
	merged := image.NewRGBA(posterBounds)
	draw.Draw(merged, posterBounds, poster, posterBounds.Min, draw.Src)
	draw.Draw(merged, image.Rectangle{Min: posterBounds.Min, Max: posterBounds.Min.Add(badge.Bounds().Size())}, badge, badge.Bounds().Min, draw.Over)

	var mergedData bytes.Buffer
	if err := jpeg.Encode(&mergedData, merged, nil); err != nil {
		return "", fmt.Errorf("生成封面图片失败: %w", err)
	}
	return s.service.UploadImage(mergedData.Bytes(), "poster.jpg", "poster")
}

func posterBadgeName(keyword string) string {
	keyword = strings.ToUpper(keyword)
	switch {
	case strings.HasSuffix(keyword, "-UC"):
		return "uc.png"
	case strings.HasSuffix(keyword, "-CH"), strings.HasSuffix(keyword, "-C"):
		return "c.png"
	case strings.HasSuffix(keyword, "-U"):
		return "u.png"
	default:
		return ""
	}
}

func scaleToFit(src image.Image, maxWidth, maxHeight int) image.Image {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= maxWidth && height <= maxHeight {
		return src
	}
	scale := float64(maxWidth) / float64(width)
	if height*maxWidth > width*maxHeight {
		scale = float64(maxHeight) / float64(height)
	}
	newWidth, newHeight := int(float64(width)*scale), int(float64(height)*scale)
	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			dst.Set(x, y, src.At(bounds.Min.X+x*width/newWidth, bounds.Min.Y+y*height/newHeight))
		}
	}
	return dst
}

func imageFilename(url string) string {
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
	return filename
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
