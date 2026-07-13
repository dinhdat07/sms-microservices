package impl

import (
	"context"
	"fmt"
	"strconv"
	monitoringDomain "sms-monitoring/internal/domain"
	infraRedis "sms-monitoring/internal/infrastructure/redis"
	"sms-monitoring/internal/repository"

	"github.com/redis/go-redis/v9"
)

// RedisServerStateStore implements domain.ServerStateStore using Redis hashes.
type RedisServerStateStore struct {
	rdb redis.UniversalClient
}

func NewRedisServerStateStore(rdb redis.UniversalClient) repository.ServerStateStore {
	return &RedisServerStateStore{rdb: rdb}
}

func (s *RedisServerStateStore) GetServerState(ctx context.Context, serverID string) (*monitoringDomain.ServerState, error) {
	redisKey := fmt.Sprintf(infraRedis.ServerInfoKeyFmt, serverID)
	vals, err := s.rdb.HGetAll(ctx, redisKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get server info from redis: %w", err)
	}
	if len(vals) == 0 {
		return nil, repository.ErrServerStateNotFound
	}
	retryCount, _ := strconv.Atoi(vals["retry_count"])
	return &monitoringDomain.ServerState{
		Status:     vals["status"],
		RetryCount: retryCount,
	}, nil
}

func (s *RedisServerStateStore) SetServerState(ctx context.Context, serverID string, status string, retryCount int) error {
	redisKey := fmt.Sprintf(infraRedis.ServerInfoKeyFmt, serverID)
	err := s.rdb.HSet(ctx, redisKey, "status", status, "retry_count", retryCount).Err()
	if err != nil {
		return fmt.Errorf("failed to update redis status: %w", err)
	}
	return nil
}
