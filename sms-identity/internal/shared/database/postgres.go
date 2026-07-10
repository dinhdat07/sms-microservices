package database

import (
	"sms-identity/internal/shared/logger"
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

		// 1. SetMaxOpenConns: Maximum number of open connections to the database.
		// Reason for 25: Safe default that can handle thousands of req/s without overwhelming the DB (e.g., Postgres default max_connections is 100).
		sqlDB.SetMaxOpenConns(25)

		// 2. SetMaxIdleConns: Maximum number of connections in the idle connection pool.
		// Reason for 25: Matching MaxOpenConns reduces latency by avoiding frequent connection creation/destruction under fluctuating loads.
		sqlDB.SetMaxIdleConns(25)

		// 3. SetConnMaxLifetime: Maximum amount of time a connection may be reused.
		// Reason for 5 minutes: Prevents "bad connection" errors caused by firewalls, load balancers, or DB timeouts silently dropping long-lived connections.
		sqlDB.SetConnMaxLifetime(5 * time.Minute)

		// 4. SetConnMaxIdleTime: Maximum amount of time a connection may be idle before being closed.
		// Reason for 3 minutes: Frees up database server resources when the system is idle.
		sqlDB.SetConnMaxIdleTime(3 * time.Minute)
	})

	return dbInstance, dbError
}