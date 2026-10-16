package scheduler

import (
	"encoding/json"
	"errors"
	"strings"

	"gorm.io/gorm"

	"fntube/internal/model"
)

type ScrapeLogEvent struct {
	Type string
	Log  model.ScrapeLog
	ID   uint
}

type scrapeStep struct {
	Step   string `json:"step"`
	Status string `json:"status"`
}

var scrapeStepProgress = map[string]struct {
	running int
	done    int
	label   string
}{
	model.StepSearch:           {running: 10, done: 20, label: "搜索影片"},
	model.StepGetDetail:        {running: 25, done: 35, label: "获取影片详情"},
	model.StepDownloadPoster:   {running: 40, done: 50, label: "下载封面"},
	model.StepDownloadBackdrop: {running: 55, done: 60, label: "下载背景图"},
	model.StepSearchActor:      {running: 65, done: 75, label: "处理演员信息"},
	model.StepTranslate:        {running: 80, done: 90, label: "翻译影片信息"},
	model.StepSave:             {running: 95, done: 98, label: "保存刮削结果"},
}

func (s *Scheduler) SubscribeScrapeLogs() (<-chan ScrapeLogEvent, func()) {
	s.scrapeSubscriberMu.Lock()
	id := s.nextScrapeSubscriber
	s.nextScrapeSubscriber++
	ch := make(chan ScrapeLogEvent, 32)
	s.scrapeSubscribers[id] = ch
	s.scrapeSubscriberMu.Unlock()

	return ch, func() {
		s.scrapeSubscriberMu.Lock()
		delete(s.scrapeSubscribers, id)
		s.scrapeSubscriberMu.Unlock()
	}
}

func (s *Scheduler) ActiveManualScrapeLogs() ([]model.ScrapeLog, error) {
	var logs []model.ScrapeLog
	err := s.db.
		Where("method = ? AND status = ?", model.ScrapeMethodManual, model.ScrapeStatusInProgress).
		Order("id desc").
		Find(&logs).Error
	if err != nil {
		return nil, err
	}
	for i := range logs {
		prepareScrapeLogProgress(&logs[i])
	}
	return logs, nil
}

func (s *Scheduler) CreateScrapeLog(logEntry *model.ScrapeLog) error {
	if logEntry.Status == "" {
		logEntry.Status = model.ScrapeStatusInProgress
	}
	if logEntry.Steps == "" {
		logEntry.Steps = "[]"
	}
	if err := s.db.Create(logEntry).Error; err != nil {
		return err
	}
	s.publishScrapeLog(*logEntry)
	return nil
}

func (s *Scheduler) DeleteScrapeLog(id uint) error {
	result := s.db.Delete(&model.ScrapeLog{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	s.publishScrapeLogEvent(ScrapeLogEvent{Type: "deleted", ID: id})
	return nil
}

func (s *Scheduler) publishScrapeLog(logEntry model.ScrapeLog) {
	prepareScrapeLogProgress(&logEntry)
	s.publishScrapeLogEvent(ScrapeLogEvent{Type: "progress", Log: logEntry, ID: logEntry.ID})
}

func (s *Scheduler) publishScrapeLogDeleted(id uint) {
	s.publishScrapeLogEvent(ScrapeLogEvent{Type: "deleted", ID: id})
}

func (s *Scheduler) publishScrapeLogEvent(event ScrapeLogEvent) {
	s.scrapeSubscriberMu.Lock()
	defer s.scrapeSubscriberMu.Unlock()
	for _, ch := range s.scrapeSubscribers {
		select {
		case ch <- event:
		default:
		}
	}
}

func (s *Scheduler) updateScrapeLog(logEntry *model.ScrapeLog, updates map[string]interface{}) error {
	if logEntry == nil || logEntry.ID == 0 {
		return errors.New("刮削记录无效")
	}
	if err := s.db.Model(logEntry).Updates(updates).Error; err != nil {
		return err
	}
	if err := s.db.First(logEntry, logEntry.ID).Error; err != nil {
		return err
	}
	s.publishScrapeLog(*logEntry)
	return nil
}

func prepareScrapeLogProgress(logEntry *model.ScrapeLog) {
	if logEntry == nil {
		return
	}
	logEntry.CurrentStep = "准备刮削"
	logEntry.Progress = 5
	var steps []scrapeStep
	if json.Unmarshal([]byte(logEntry.Steps), &steps) != nil {
		return
	}
	for _, step := range steps {
		config, ok := scrapeStepProgress[step.Step]
		if !ok {
			continue
		}
		logEntry.CurrentStep = config.label
		progress := config.running
		if step.Status != "running" {
			progress = config.done
		}
		if progress > logEntry.Progress {
			logEntry.Progress = progress
		}
	}
	switch logEntry.Status {
	case model.ScrapeStatusSuccess:
		logEntry.CurrentStep = "刮削成功"
		logEntry.Progress = 100
	case model.ScrapeStatusCompleted:
		logEntry.CurrentStep = "刮削完成"
		logEntry.Progress = 100
	case model.ScrapeStatusFailed:
		logEntry.CurrentStep = "刮削失败"
	default:
		if strings.TrimSpace(logEntry.Error) != "" {
			logEntry.CurrentStep = "刮削异常"
		}
	}
}
