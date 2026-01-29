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

	runManualMigrations(db)

	log.Println("Database initialized successfully")
	return db
}

// runManualMigrations handles schema changes that GORM's AutoMigrate can't do,
// like dropping NOT NULL constraints in SQLite (which doesn't support ALTER COLUMN).
// Each migration is idempotent, so it's safe to run on every startup.
func runManualMigrations(db *gorm.DB) {
	// Make google_id nullable for local (non-Google) users.
	// SQLite requires recreating the table to change column constraints.
	// We check if the column is still NOT NULL before doing the migration.
	var notNull bool
	row := db.Raw(`SELECT "notnull" FROM pragma_table_info('users') WHERE name = 'google_id'`).Row()
	if err := row.Scan(&notNull); err != nil {
		// Column doesn't exist or table doesn't exist yet, nothing to migrate
		return
	}
	if !notNull {
		return // Already nullable
	}

	log.Println("Migrating: making google_id nullable for local auth support")
	statements := []string{
		`CREATE TABLE users_backup (
			id TEXT PRIMARY KEY,
			google_id TEXT,
			email TEXT NOT NULL,
			name TEXT NOT NULL,
			avatar_url TEXT,
			password_hash TEXT,
			created_at DATETIME
		)`,
		`INSERT INTO users_backup SELECT id, google_id, email, name, avatar_url, COALESCE(password_hash, ''), created_at FROM users`,
		`DROP TABLE users`,
		`ALTER TABLE users_backup RENAME TO users`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
	}

	tx := db.Begin()
	for _, stmt := range statements {
		if err := tx.Exec(stmt).Error; err != nil {
			tx.Rollback()
			log.Fatalf("Migration failed on statement [%s]: %v", stmt, err)
		}
	}
	if err := tx.Commit().Error; err != nil {
		log.Fatalf("Migration commit failed: %v", err)
	}
	log.Println("Migration complete: google_id is now nullable")
}
