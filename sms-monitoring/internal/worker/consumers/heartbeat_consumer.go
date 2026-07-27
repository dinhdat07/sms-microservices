package consumers

import (
	"context"
	"fmt"

	"sms-monitoring/internal/infrastructure/logger"
	"sms-monitoring/internal/infrastructure/messagebroker"
	infraRedis "sms-monitoring/internal/infrastructure/redis"
	"sms-monitoring/internal/service"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type HeartbeatConsumer struct {
	subscriber messagebroker.Subscriber
	monService service.MonitoringService
	rdb        redis.UniversalClient
	stream     string
	group      string
	consumer   string
}

func NewHeartbeatConsumer(subscriber messagebroker.Subscriber, monService service.MonitoringService, rdb redis.UniversalClient) *HeartbeatConsumer {
	return &HeartbeatConsumer{
		subscriber: subscriber,
		monService: monService,
		rdb:        rdb,
		stream:     "sms.events.heartbeat",
		group:      "monitoring_group",
		consumer:   "monitoring_worker_heartbeat",
	}
}

func (c *HeartbeatConsumer) Start(ctx context.Context) {
	if c.rdb == nil {
		logger.Log.Warn("Redis not enabled, heartbeat consumer disabled.")
		return
	}

	logger.Log.Info("Starting Heartbeat Consumer", zap.String("stream", c.stream))

	go func() {
		err := c.subscriber.Subscribe(ctx, c.stream, c.group, c.consumer, c.processMessage)
		if err != nil {
			logger.Log.Error("Heartbeat consumer failed", zap.Error(err))
		}
	}()
}

func (c *HeartbeatConsumer) processMessage(ctx context.Context, msg messagebroker.Message) error {
	serverID, ok := msg.Values["server_id"].(string)
	if !ok || serverID == "" {
		return nil
	}

	// Get IP from Hash
	redisKey := fmt.Sprintf(infraRedis.ServerInfoKeyFmt, serverID)
	ipv4, err := c.rdb.HGet(ctx, redisKey, infraRedis.ServerInfoFieldIPv4).Result()
	if err != nil {
		ipv4 = "" // Fallback
	}

	// Evaluate as UP
	if err := c.monService.Evaluate(ctx, serverID, ipv4, true); err != nil {
		logger.Log.Error("Failed to evaluate state for heartbeat", zap.Error(err), zap.String("server_id", serverID))
		return err
	}

	return nil
}
