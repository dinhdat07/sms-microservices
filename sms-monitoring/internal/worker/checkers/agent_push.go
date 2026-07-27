package checkers

import (
	"context"
	"time"

	infraRedis "sms-monitoring/internal/infrastructure/redis"

	"github.com/redis/go-redis/v9"
)

type AgentPushChecker struct {
	rdb redis.UniversalClient
	ttl float64
}

func NewAgentPushChecker(rdb redis.UniversalClient, ttlSecs float64) HealthChecker {
	return &AgentPushChecker{rdb: rdb, ttl: ttlSecs}
}

func (c *AgentPushChecker) Check(ctx context.Context, config ServerConfig) bool {
	serverID := config["server_id"]
	if serverID == "" {
		return false
	}

	// Check the ZSET for the last heartbeat score (timestamp)
	score, err := c.rdb.ZScore(ctx, infraRedis.AgentHeartbeatZSetKey, serverID).Result()
	if err != nil {
		if err == redis.Nil {
			return false // No heartbeat received yet
		}
		return false
	}

	// If heartbeat is older than TTL, consider offline
	if float64(time.Now().Unix())-score > c.ttl {
		return false
	}

	return true
}
