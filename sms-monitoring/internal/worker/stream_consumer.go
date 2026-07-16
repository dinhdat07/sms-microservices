package worker

import (
	"context"
	"encoding/json"
	"fmt"

	infraRedis "sms-monitoring/internal/infrastructure/redis"
	"sms-monitoring/internal/infrastructure/logger"
	"sms-monitoring/internal/infrastructure/messagebroker"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// StreamConsumer listens to the Outbox events on Redis Stream
type StreamConsumer struct {
	subscriber messagebroker.Subscriber
	rdb        redis.UniversalClient
	stream     string
	group      string
	consumer   string
}

func NewStreamConsumer(subscriber messagebroker.Subscriber, rdb redis.UniversalClient) *StreamConsumer {
	return &StreamConsumer{
		subscriber: subscriber,
		rdb:        rdb,
		stream:     "sms.events.server",
		group:      "monitoring_group",
		consumer:   "monitoring_worker_1",
	}
}

func (c *StreamConsumer) Start(ctx context.Context) {
	if c.rdb == nil {
		logger.Log.Warn("Redis not enabled, stream consumer disabled.")
		return
	}

	logger.Log.Info("Starting Stream Consumer", zap.String("stream", c.stream))

	go func() {
		err := c.subscriber.Subscribe(ctx, c.stream, c.group, c.consumer, c.processMessage)
		if err != nil {
			logger.Log.Error("Stream consumer failed", zap.Error(err))
		}
	}()
}

func (c *StreamConsumer) processMessage(ctx context.Context, msg messagebroker.Message) error {
	eventType, ok := msg.Values["event_type"].(string)
	if !ok {
		return nil
	}

	payloadStr, ok := msg.Values["payload"].(string)
	if !ok {
		return nil
	}

	var payload struct {
		ID   string `json:"server_id"`
		IPv4 string `json:"ipv4"`
	}

	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		logger.Log.Error("Failed to unmarshal event payload", zap.Error(err))
		return err
	}

	serverID := payload.ID
	ipv4 := payload.IPv4

	switch eventType {
	case "ServerCreated", "ServerUpdated":
		logger.Log.Info("Syncing Server to Monitoring Cache", zap.String("event", eventType), zap.String("id", serverID))
		c.rdb.SAdd(ctx, infraRedis.ServerAllIDsKey, serverID)
		
		if ipv4 != "" {
			c.rdb.HSet(ctx, fmt.Sprintf(infraRedis.ServerInfoKeyFmt, serverID), "ipv4", ipv4)
		}

	case "ServerDeleted":
		logger.Log.Info("Removing Server from Monitoring Cache", zap.String("id", serverID))
		c.rdb.SRem(ctx, infraRedis.ServerAllIDsKey, serverID)
		c.rdb.Del(ctx, fmt.Sprintf(infraRedis.ServerInfoKeyFmt, serverID))
	}

	return nil
}
