package database

import (
	"sms-reporting/internal/infrastructure/logger"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	dbInstance *gorm.DB
	dbOnce     sync.Once
	dbError    error
)

// GetInstance returns a Singleton Database Connection
func GetInstance(dsn string) (*gorm.DB, error) {
	dbOnce.Do(func() {
		dbInstance, dbError = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if dbError != nil {
			logger.Log.Sugar().Errorf("Failed to connect to database: %v", dbError)
			return
		}

		// Retrieve the underlying sql.DB to configure the Connection Pool
		sqlDB, err := dbInstance.DB()
		if err != nil {
			dbError = err
			return
		}

		sqlDB.SetMaxOpenConns(25)
		sqlDB.SetMaxIdleConns(25)
		sqlDB.SetConnMaxLifetime(5 * time.Minute)
		sqlDB.SetConnMaxIdleTime(3 * time.Minute)
	})

	return dbInstance, dbError
}
