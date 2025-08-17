package database

import (
	"fmt"
	"time"

	"tiris-backend/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DB struct {
	*gorm.DB
}

func Initialize(cfg config.DatabaseConfig) (*DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.Username,
		cfg.Password,
		cfg.DatabaseName,
		cfg.SSLMode,
	)

	// Configure GORM logger
	var logLevel logger.LogLevel
	if cfg.SSLMode == "disable" { // Assume development environment
		logLevel = logger.Info
	} else {
		logLevel = logger.Error
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL DB to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying SQL DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Second)

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable TimescaleDB extension (optional in test environments)
	if err := enableTimescaleDB(db); err != nil {
		// In test environments, TimescaleDB might not be available
		// Log the warning but continue
		fmt.Printf("Warning: TimescaleDB not available: %v\n", err)
	}

	return &DB{db}, nil
}

func enableTimescaleDB(db *gorm.DB) error {
	// Check if TimescaleDB extension exists
	var extensionExists bool
	err := db.Raw("SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'timescaledb')").Scan(&extensionExists).Error
	if err != nil {
		// In test environments, this query might fail if TimescaleDB is not available
		fmt.Printf("Warning: Unable to check TimescaleDB extension availability: %v\n", err)
		return nil
	}

	// Create extension if it doesn't exist (requires superuser privileges)
	if !extensionExists {
		err = db.Exec("CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE").Error
		if err != nil {
			// Log warning but don't fail if we can't create the extension
			// This is expected in test environments and managed database environments
			fmt.Printf("Info: TimescaleDB extension not available (normal in test environments): %v\n", err)
		} else {
			fmt.Println("Info: TimescaleDB extension enabled successfully")
		}
	} else {
		fmt.Println("Info: TimescaleDB extension already enabled")
	}

	return nil
}

func Close(db *DB) error {
	if db == nil {
		return nil
	}

	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying SQL DB: %w", err)
	}

	return sqlDB.Close()
}

func (db *DB) HealthCheck() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying SQL DB: %w", err)
	}

	return sqlDB.Ping()
}
