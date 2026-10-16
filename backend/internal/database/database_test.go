package database

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"fntube/internal/model"
)

func TestInitMigratesLegacyCloudJobSmartStrmColumns(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "fntube.db")
	legacy, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open legacy database: %v", err)
	}
	if err := legacy.Exec(`CREATE TABLE guangya_cloud_add_jobs (
		id integer PRIMARY KEY AUTOINCREMENT,
		task_id text NOT NULL,
		requested_name text NOT NULL,
		selected_files text NOT NULL,
		status text NOT NULL
	)`).Error; err != nil {
		t.Fatalf("create legacy cloud job table: %v", err)
	}
	sqlDB, err := legacy.DB()
	if err != nil {
		t.Fatalf("get legacy sql database: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close legacy database: %v", err)
	}

	db, err := Init(dbPath)
	if err != nil {
		t.Fatalf("migrate legacy database: %v", err)
	}
	migratedSQLDB, err := db.DB()
	if err != nil {
		t.Fatalf("get migrated sql database: %v", err)
	}
	defer migratedSQLDB.Close()

	for _, column := range []string{
		"smart_strm_task_name",
		"smart_strm_storage_root",
		"smart_strm_storage_path",
		"smart_strm_notified",
		"smart_strm_notified_at",
	} {
		if !db.Migrator().HasColumn(&model.GuangyaCloudAddJob{}, column) {
			t.Fatalf("migrated cloud job table is missing column %q", column)
		}
	}

	job := model.GuangyaCloudAddJob{
		TaskID:        "legacy-migration",
		RequestedName: "TEST-001",
		SelectedFiles: "[]",
		Status:        "pending",
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create migrated cloud job: %v", err)
	}
	if err := db.Model(&job).Updates(map[string]any{
		"smart_strm_notified":     true,
		"smart_strm_storage_path": "/日本/TEST-001",
	}).Error; err != nil {
		t.Fatalf("update migrated SmartStrm columns: %v", err)
	}
}
