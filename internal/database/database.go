package database

import (
	"fmt"
	"time"

	"github.com/digi-con/hackathon-template/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DB interface {
	GetDB() *gorm.DB
	Migrate() error
	Health() error
	Close() error
}

type database struct {
	db *gorm.DB
}

func Initialize(cfg *config.Config) (DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.DB.Host,
		cfg.DB.User,
		cfg.DB.Password,
		cfg.DB.Name,
		cfg.DB.Port,
		cfg.DB.SSLMode,
	)

	var db *gorm.DB
	var err error
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(getLogLevel(cfg.LogLevel)),
		})
		if err == nil {
			break
		}
		fmt.Printf("DB connect failed: %v (try %d/%d)\n", err, i+1, maxRetries)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after retries: %w", err)
	}

	// ... existing code (from line 52 onward) ...
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &database{db: db}

	// Auto-migrate tables
	if err := database.Migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return database, nil
}

func (d *database) GetDB() *gorm.DB {
	return d.db
}

func (d *database) Migrate() error {
	return d.db.AutoMigrate(
		&User{},
		&File{},
	)
}

func (d *database) Health() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

func (d *database) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
} 

// 補助関数
func getLogLevel(logLevel string) logger.LogLevel {
	switch logLevel {
	case "debug":
		return logger.Info
	case "error":
		return logger.Error
	default:
		return logger.Warn
	}
}