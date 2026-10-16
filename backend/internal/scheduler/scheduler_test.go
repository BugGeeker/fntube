package scheduler

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"fntube/internal/metatube"
	"fntube/internal/model"
	"fntube/internal/trimmedia"
)

func newTestScheduler(t *testing.T) (*Scheduler, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "fntube.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get test database connection: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})
	if err := db.AutoMigrate(&model.ScrapeLog{}); err != nil {
		t.Fatalf("migrate scrape logs: %v", err)
	}

	service := trimmedia.NewService("", "", "", "", "", nil)
	return NewScheduler(db, service), db
}

func TestScrapeSingleRecordsPreflightFailure(t *testing.T) {
	sched, db := newTestScheduler(t)

	_, err := sched.ScrapeSingle("item-guid")
	if err == nil {
		t.Fatal("ScrapeSingle() error = nil, want preflight error")
	}

	var logEntry model.ScrapeLog
	if err := db.Where("item_guid = ?", "item-guid").First(&logEntry).Error; err != nil {
		t.Fatalf("find scrape log: %v", err)
	}
	if logEntry.Method != model.ScrapeMethodManual {
		t.Fatalf("method = %q, want %q", logEntry.Method, model.ScrapeMethodManual)
	}
	if logEntry.Status != model.ScrapeStatusFailed {
		t.Fatalf("status = %q, want %q", logEntry.Status, model.ScrapeStatusFailed)
	}
	if !strings.Contains(logEntry.Error, "飞牛影视未连接") {
		t.Fatalf("error = %q, want connection failure", logEntry.Error)
	}
}

func TestStartScrapeSingleCreatesRecordBeforeReturning(t *testing.T) {
	sched, db := newTestScheduler(t)
	oldLog := model.ScrapeLog{
		ItemGUID: "async-item-guid",
		Method:   model.ScrapeMethodManual,
		Status:   model.ScrapeStatusSuccess,
		Steps:    "[]",
	}
	if err := db.Create(&oldLog).Error; err != nil {
		t.Fatalf("create old scrape log: %v", err)
	}

	if err := sched.StartScrapeSingle("async-item-guid"); err != nil {
		t.Fatalf("StartScrapeSingle() error = %v", err)
	}

	var logEntry model.ScrapeLog
	if err := db.Where("item_guid = ?", "async-item-guid").First(&logEntry).Error; err != nil {
		t.Fatalf("scrape log was not created before return: %v", err)
	}
	if logEntry.ID == oldLog.ID {
		t.Fatalf("old scrape log %d was not replaced", oldLog.ID)
	}
	var count int64
	if err := db.Model(&model.ScrapeLog{}).Where("item_guid = ?", "async-item-guid").Count(&count).Error; err != nil {
		t.Fatalf("count scrape logs: %v", err)
	}
	if count != 1 {
		t.Fatalf("scrape log count = %d, want 1", count)
	}

	deadline := time.Now().Add(time.Second)
	for logEntry.Status != model.ScrapeStatusFailed && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
		if err := db.First(&logEntry, logEntry.ID).Error; err != nil {
			t.Fatalf("refresh scrape log: %v", err)
		}
	}
	if logEntry.Status != model.ScrapeStatusFailed {
		t.Fatalf("status = %q, want %q", logEntry.Status, model.ScrapeStatusFailed)
	}
}

func TestScrapeLogEventsPublishProgressAndDeletion(t *testing.T) {
	sched, _ := newTestScheduler(t)
	events, unsubscribe := sched.SubscribeScrapeLogs()
	defer unsubscribe()

	logEntry, err := sched.createScrapeLog("event-item-guid", "EVENT-001", model.ScrapeMethodManual, 0)
	if err != nil {
		t.Fatalf("create scrape log: %v", err)
	}

	select {
	case event := <-events:
		if event.Type != "progress" || event.Log.ID != logEntry.ID {
			t.Fatalf("unexpected create event: %#v", event)
		}
		if event.Log.CurrentStep != "准备刮削" || event.Log.Progress != 5 {
			t.Fatalf("unexpected initial progress: %#v", event.Log)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for create event")
	}

	steps, err := json.Marshal([]map[string]string{{
		"step": model.StepSearch, "status": "running",
	}})
	if err != nil {
		t.Fatalf("marshal steps: %v", err)
	}
	if err := sched.updateScrapeLog(logEntry, map[string]interface{}{"steps": string(steps)}); err != nil {
		t.Fatalf("update scrape log: %v", err)
	}

	select {
	case event := <-events:
		if event.Log.CurrentStep != "搜索影片" || event.Log.Progress != 10 {
			t.Fatalf("unexpected step progress: %#v", event.Log)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for progress event")
	}

	if err := sched.DeleteScrapeLog(logEntry.ID); err != nil {
		t.Fatalf("delete scrape log: %v", err)
	}
	select {
	case event := <-events:
		if event.Type != "deleted" || event.ID != logEntry.ID {
			t.Fatalf("unexpected delete event: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for delete event")
	}
}

func TestResolveScrapeKeyword(t *testing.T) {
	tests := []struct {
		name       string
		item       trimmedia.MediaServerItem
		streamList *trimmedia.StreamListResult
		want       string
		wantSource string
	}{
		{
			name: "prefer filename without extension",
			item: trimmedia.MediaServerItem{
				Title:         "影片标题",
				OriginalTitle: "Original Title",
			},
			streamList: &trimmedia.StreamListResult{
				Files: []trimmedia.MediaFileInfo{{FileName: "ABC-123.mp4"}},
			},
			want:       "ABC-123",
			wantSource: "file_name",
		},
		{
			name: "fallback to title when stream list is unavailable",
			item: trimmedia.MediaServerItem{
				Title:         "影片标题",
				OriginalTitle: "Original Title",
			},
			want:       "影片标题",
			wantSource: "title",
		},
		{
			name: "fallback to original title",
			item: trimmedia.MediaServerItem{
				Title:         " ",
				OriginalTitle: "Original Title",
			},
			streamList: &trimmedia.StreamListResult{},
			want:       "Original Title",
			wantSource: "original_title",
		},
		{
			name:       "no keyword",
			item:       trimmedia.MediaServerItem{},
			streamList: &trimmedia.StreamListResult{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotSource := resolveScrapeKeyword(tt.item, tt.streamList)
			if got != tt.want {
				t.Fatalf("keyword = %q, want %q", got, tt.want)
			}
			if gotSource != tt.wantSource {
				t.Fatalf("source = %q, want %q", gotSource, tt.wantSource)
			}
		})
	}
}

func TestNormalizeMovieNumber(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "YMDD-284", want: "YMDD-284"},
		{input: "YMDD-284-C", want: "YMDD-284"},
		{input: "YMDD-284-1", want: "YMDD-284"},
		{input: "ymdd_284_c", want: "YMDD-284"},
		{input: "[javdb] YMDD-284-C", want: "YMDD-284"},
		{input: "N1901", want: "N1901"},
		{input: "N1901-C", want: "N1901"},
		{input: "N1901-1", want: "N1901"},
		{input: "N1901-U", want: "N1901"},
		{input: "N1901-UC", want: "N1901"},
		{input: "YMDD-284-UC", want: "YMDD-284"},
		{input: "SSNI-311-S1", want: "SSNI-311"},
		{input: "SSNI-311-C", want: "SSNI-311"},
		{input: "RHJ-139 Red Hot Jam Vol.139 制服美女俱乐部", want: "RHJ-139"},
		{input: "FC2-PPV-123456-1", want: "FC2-PPV-123456"},
		{input: "not-a-number", want: ""},
		{input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := normalizeMovieNumber(tt.input); got != tt.want {
				t.Fatalf("normalizeMovieNumber(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMatchingMovieResult(t *testing.T) {
	results := []metatube.MovieSearchResult{
		{ID: "wrong", Number: "NAGAE2-0588", Provider: "provider-a"},
		{ID: "matched", Number: "YMDD-284", Provider: "provider-b"},
	}

	for _, keyword := range []string{"YMDD-284", "YMDD-284-C", "YMDD-284-1"} {
		t.Run(keyword, func(t *testing.T) {
			got, ok := matchingMovieResult(keyword, results)
			if !ok {
				t.Fatalf("matchingMovieResult(%q) found no result", keyword)
			}
			if got.ID != "matched" {
				t.Fatalf("matchingMovieResult(%q) ID = %q, want matched", keyword, got.ID)
			}
		})
	}
}

func TestMatchingMovieResultRejectsMismatch(t *testing.T) {
	results := []metatube.MovieSearchResult{
		{ID: "wrong", Number: "NAGAE2-0588"},
		{ID: "empty", Number: ""},
	}

	if got, ok := matchingMovieResult("YMDD-284-C", results); ok {
		t.Fatalf("matchingMovieResult() = %#v, want no match", got)
	}
}

func TestMatchingMovieResultKeepsTitleSearchFallback(t *testing.T) {
	results := []metatube.MovieSearchResult{
		{ID: "first", Number: "YMDD-284"},
	}

	got, ok := matchingMovieResult("影片标题", results)
	if !ok || got.ID != "first" {
		t.Fatalf("matchingMovieResult() = %#v, %v, want first result", got, ok)
	}
}
