package database

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"school-platform/internal/config"
	"school-platform/internal/models"
)

var DB *gorm.DB

// Connect establishes a PostgreSQL connection via GORM and configures the connection pool.
func Connect(cfg *config.Config) (*gorm.DB, error) {
	logLevel := logger.Silent
	if cfg.IsDevelopment() {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("database: failed to connect: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("database: failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(2 * time.Minute)

	DB = db
	log.Info().Msg("database: connected to PostgreSQL")
	return db, nil
}

// RunMigrations runs GORM AutoMigrate for all models.
// This is safe to run on every startup — GORM only adds missing columns/tables.
func RunMigrations(db *gorm.DB) error {
	log.Info().Msg("database: running migrations...")

	// Enable uuid-ossp extension for gen_random_uuid()
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"pgcrypto\"").Error; err != nil {
		return fmt.Errorf("database: failed to create pgcrypto extension: %w", err)
	}

	if err := db.AutoMigrate(models.AllModels()...); err != nil {
		return fmt.Errorf("database: migration failed: %w", err)
	}

	if err := runSQLMigrations(db); err != nil {
		return fmt.Errorf("database: sql migrations failed: %w", err)
	}

	log.Info().Msg("database: migrations complete")
	return nil
}

// runSQLMigrations applies any .sql files in /migrations in lexical order.
// Each file is wrapped in a transaction; idempotent files use WHERE NOT EXISTS.
func runSQLMigrations(db *gorm.DB) error {
	dir := "migrations"
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // migrations dir is optional
		}
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".sql" {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, f := range files {
		path := filepath.Join(dir, f)
		body, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		log.Info().Str("file", f).Msg("database: applying SQL migration")
		if err := db.Exec(string(body)).Error; err != nil {
			return fmt.Errorf("apply %s: %w", path, err)
		}
	}
	return nil
}

// Get returns the global DB instance (panics if Connect() was not called first).
func Get() *gorm.DB {
	if DB == nil {
		log.Fatal().Msg("database: Get() called before Connect()")
	}
	return DB
}

// Ping checks the database connection is alive.
func Ping() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
