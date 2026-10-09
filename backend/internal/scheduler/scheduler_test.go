package scheduler

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

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
