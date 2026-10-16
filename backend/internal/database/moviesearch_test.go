package database

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"fntube/internal/model"
)

func TestMovieSearchConfigMigrationAddsDomainColumns(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "fntube.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get database connection: %v", err)
	}
	defer sqlDB.Close()

	if err := db.Exec(`
		CREATE TABLE movie_search_configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source TEXT NOT NULL DEFAULT "javbus",
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	if err := db.AutoMigrate(&model.MovieSearchConfig{}); err != nil {
		t.Fatalf("migrate movie search config: %v", err)
	}

	for _, column := range []string{"jav_db_domain", "jav_bus_domain"} {
		if !db.Migrator().HasColumn(&model.MovieSearchConfig{}, column) {
			t.Fatalf("expected migrated column %q", column)
		}
	}

	config := model.MovieSearchConfig{
		Source:       "javdb",
		JavDBDomain:  "db.example.com",
		JavBusDomain: "bus.example.com",
	}
	if err := db.Create(&config).Error; err != nil {
		t.Fatalf("save migrated config: %v", err)
	}

	var saved model.MovieSearchConfig
	if err := db.First(&saved, config.ID).Error; err != nil {
		t.Fatalf("read migrated config: %v", err)
	}
	if saved.JavDBDomain != config.JavDBDomain || saved.JavBusDomain != config.JavBusDomain {
		t.Fatalf("unexpected saved domains: %#v", saved)
	}
}
