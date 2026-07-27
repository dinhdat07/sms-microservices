package checkers

import (
	"context"
	"time"

	infraRedis "sms-monitoring/internal/infrastructure/redis"

	"github.com/redis/go-redis/v9"
)

type AgentPushChecker struct {
	rdb redis.UniversalClient
}

func NewAgentPushChecker(rdb redis.UniversalClient) HealthChecker {
	return &AgentPushChecker{rdb: rdb}
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

	// If heartbeat is older than 60 seconds, consider offline
	if float64(time.Now().Unix())-score > 60 {
		return false
	}

	return true
}
