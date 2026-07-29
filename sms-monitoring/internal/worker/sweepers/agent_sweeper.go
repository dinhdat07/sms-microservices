package sweepers

import (
	"context"
	"fmt"
	"time"

	"sms-monitoring/internal/infrastructure/logger"
	infraRedis "sms-monitoring/internal/infrastructure/redis"
	"sms-monitoring/internal/service"

	"github.com/redis/go-redis/v9"
)

type AgentSweeper struct {
	rdb        redis.UniversalClient
	monService service.MonitoringService
	interval   time.Duration
	timeoutSec int64
}

func NewAgentSweeper(rdb redis.UniversalClient, monService service.MonitoringService, interval time.Duration, timeoutSec int64) *AgentSweeper {
	return &AgentSweeper{
		rdb:        rdb,
		monService: monService,
		interval:   interval,
		timeoutSec: timeoutSec,
	}
}

func (s *AgentSweeper) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	logger.Log.Sugar().Infof("[AgentSweeper] Started with interval %v, timeout %ds", s.interval, s.timeoutSec)

	for {
		select {
		case <-ctx.Done():
			logger.Log.Sugar().Info("[AgentSweeper] Stopped")
			return
		case <-ticker.C:
			s.sweep(ctx)
		}
	}
}

func (s *AgentSweeper) sweep(ctx context.Context) {
	now := time.Now().Unix()
	expiredScore := now - s.timeoutSec

	// Get all members with score <= expiredScore
	expiredServers, err := s.rdb.ZRangeByScore(ctx, infraRedis.AgentHeartbeatZSetKey, &redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%d", expiredScore),
	}).Result()

	if err != nil && err != redis.Nil {
		logger.Log.Sugar().Errorf("[AgentSweeper] Failed to ZRangeByScore: %v", err)
		return
	}

	for _, serverID := range expiredServers {
		logger.Log.Sugar().Infof("[AgentSweeper] Server %s agent heartbeat expired", serverID)

		// Get IP from Hash
		redisKey := fmt.Sprintf(infraRedis.ServerInfoKeyFmt, serverID)
		ipv4, err := s.rdb.HGet(ctx, redisKey, infraRedis.ServerInfoFieldIPv4).Result()
		if err != nil {
			ipv4 = ""
		}

		// Evaluate as failed
		if err := s.monService.Evaluate(ctx, serverID, ipv4, false); err != nil {
			logger.Log.Sugar().Errorf("[AgentSweeper] Failed to evaluate state for server %s: %v", serverID, err)
		}

		// Remove from ZSet so we don't keep evaluating it every 5s
		s.rdb.ZRem(ctx, infraRedis.AgentHeartbeatZSetKey, serverID)
	}
}
