package storage

import (
	"log"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/codyseavey/bets/models"
)

func InitDB(dbPath string) *gorm.DB {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.Group{},
		&models.GroupMember{},
		&models.Pool{},
		&models.PoolOption{},
		&models.Bet{},
		&models.PointsLog{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	log.Println("Database initialized successfully")
	return db
}
