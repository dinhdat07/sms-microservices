package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"sms-monitoring/internal/infrastructure/logger"
	infraRedis "sms-monitoring/internal/infrastructure/redis"

	"github.com/redis/go-redis/v9"
)

// StreamConsumer listens to the Outbox events on Redis Stream
type StreamConsumer struct {
	rdb    redis.UniversalClient
	stream string
	group  string
}

func NewStreamConsumer(rdb redis.UniversalClient) *StreamConsumer {
	return &StreamConsumer{
		rdb:    rdb,
		stream: "sms.events.server",
		group:  "monitoring-group",
	}
}

func (c *StreamConsumer) Start(ctx context.Context) {
	// Create consumer group
	err := c.rdb.XGroupCreateMkStream(ctx, c.stream, c.group, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		logger.Log.Sugar().Errorf("Failed to create consumer group: %v", err)
	}

	logger.Log.Sugar().Infof("Started monitoring stream consumer for stream %s, group %s", c.stream, c.group)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Read pending messages or new messages
		args := &redis.XReadGroupArgs{
			Group:    c.group,
			Consumer: "worker-1",
			Streams:  []string{c.stream, ">"},
			Count:    10,
			Block:    2 * time.Second,
		}

		streams, err := c.rdb.XReadGroup(ctx, args).Result()
		if err != nil {
			if err != redis.Nil {
				logger.Log.Sugar().Errorf("Error reading from stream: %v", err)
				time.Sleep(2 * time.Second)
			}
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				c.processMessage(ctx, msg)
				c.rdb.XAck(ctx, c.stream, c.group, msg.ID)
			}
		}
	}
}

func (c *StreamConsumer) processMessage(ctx context.Context, msg redis.XMessage) {
	payloadStr, ok := msg.Values["payload"].(string)
	if !ok {
		logger.Log.Sugar().Errorf("Invalid payload format in message %s", msg.ID)
		return
	}

	eventType, ok := msg.Values["event_type"].(string)
	if !ok {
		logger.Log.Sugar().Errorf("Invalid event_type format in message %s", msg.ID)
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		logger.Log.Sugar().Errorf("Failed to unmarshal payload: %v", err)
		return
	}

	serverID, _ := payload["id"].(string)
	if serverID == "" {
		return
	}

	logger.Log.Sugar().Infof("Received event %s for server %s", eventType, serverID)

	switch eventType {
	case "ServerCreated", "ServerUpdated":
		c.rdb.SAdd(ctx, infraRedis.ServerAllIDsKey, serverID)
		// Update IPv4 in cache so pinger can use it
		ipv4, _ := payload["ipv4"].(string)
		if ipv4 != "" {
			c.rdb.HSet(ctx, fmt.Sprintf(infraRedis.ServerInfoKeyFmt, serverID), "ipv4", ipv4)
		}
	case "ServerDeleted":
		c.rdb.SRem(ctx, infraRedis.ServerAllIDsKey, serverID)
		c.rdb.Del(ctx, fmt.Sprintf(infraRedis.ServerInfoKeyFmt, serverID))
	}
}
