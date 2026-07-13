package database

import (
	"context"
	"time"

	"sms-management/internal/shared/config"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(cfg *config.RedisConfig) *redis.Client {
	if cfg == nil || !cfg.Enabled {
		return nil
	}

	return redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
	})
}

func PingRedis(ctx context.Context, rdb redis.UniversalClient) error {
	if rdb == nil {
		return nil
	}
	return rdb.Ping(ctx).Err()
}

// AcquireLock attempts to acquire a distributed lock in Redis, no need ReleaseLock because of long TTL configuration
func AcquireLock(ctx context.Context, rdb redis.UniversalClient, key string, expiration time.Duration) (bool, error) {
	if rdb == nil {
		return false, nil
	}
	return rdb.SetNX(ctx, key, "locked", expiration).Result()
}
