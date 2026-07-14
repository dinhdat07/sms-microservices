package consumer

import (
	"context"
	"encoding/json"
	"time"

	"sms-reporting/internal/domain"
	"sms-reporting/internal/infrastructure/logger"
	"sms-reporting/internal/repository"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type EventConsumer interface {
	Start(ctx context.Context)
}

type eventConsumerImpl struct {
	rdb  redis.UniversalClient
	repo repository.ReportingRepository
}

func NewEventConsumer(rdb redis.UniversalClient, repo repository.ReportingRepository) EventConsumer {
	return &eventConsumerImpl{
		rdb:  rdb,
		repo: repo,
	}
}

func (c *eventConsumerImpl) Start(ctx context.Context) {
	if c.rdb == nil {
		logger.Log.Warn("[EventConsumer] Redis not configured, skipping consumer.")
		return
	}

	logger.Log.Info("[EventConsumer] Starting Redis stream consumer")

	// Create Consumer Group for Server Events
	c.createConsumerGroup(ctx, "sms.events.server", "reporting_server_group")
	
	// Create Consumer Group for Status Events
	c.createConsumerGroup(ctx, "sms.events.server_status", "reporting_status_group")

	go c.consumeStream(ctx, "sms.events.server", "reporting_server_group", "consumer_1", c.handleServerEvent)
	go c.consumeStream(ctx, "sms.events.server_status", "reporting_status_group", "consumer_1", c.handleStatusEvent)
}

func (c *eventConsumerImpl) createConsumerGroup(ctx context.Context, stream string, group string) {
	err := c.rdb.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil {
		if err.Error() == "BUSYGROUP Consumer Group name already exists" {
			// ignore
		} else {
			logger.Log.Error("Failed to create consumer group", zap.String("stream", stream), zap.Error(err))
		}
	}
}

func (c *eventConsumerImpl) consumeStream(ctx context.Context, stream, group, consumer string, handler func(context.Context, redis.XMessage) error) {
	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("Stopping consumer", zap.String("stream", stream))
			return
		default:
			// Read events
			res, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    group,
				Consumer: consumer,
				Streams:  []string{stream, ">"},
				Count:    10,
				Block:    5 * time.Second,
			}).Result()

			if err != nil {
				if err == redis.Nil {
					// Timeout, just continue
					continue
				}
				logger.Log.Error("Error reading from stream", zap.String("stream", stream), zap.Error(err))
				time.Sleep(2 * time.Second)
				continue
			}

			for _, xStream := range res {
				for _, msg := range xStream.Messages {
					err := handler(ctx, msg)
					if err != nil {
						logger.Log.Error("Failed to handle message", zap.String("id", msg.ID), zap.Error(err))
						// We don't ACK if error, let it go to Pending/DLQ in future
					} else {
						c.rdb.XAck(ctx, stream, group, msg.ID)
					}
				}
			}
		}
	}
}

func (c *eventConsumerImpl) handleServerEvent(ctx context.Context, msg redis.XMessage) error {
	eventType, ok := msg.Values["event_type"].(string)
	if !ok {
		return nil
	}

	payloadStr, ok := msg.Values["payload"].(string)
	if !ok {
		return nil
	}

	switch eventType {
	case "ServerAdded", "ServerUpdated":
		var payload struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			IP   string `json:"ip"`
		}
		if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
			return err
		}

		logger.Log.Info("Received ServerAdded/Updated event, upserting local DB", zap.String("server_id", payload.ID))
		return c.repo.UpsertReportingServer(ctx, &domain.ReportingServer{
			ServerID: payload.ID,
			Name:     payload.Name,
			IPv4:     payload.IP,
		})

	case "ServerDeleted":
		var payload struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
			return err
		}
		
		logger.Log.Info("Received ServerDeleted event, removing from local DB", zap.String("server_id", payload.ID))
		return c.repo.DeleteReportingServer(ctx, payload.ID)
	}

	return nil
}

func (c *eventConsumerImpl) handleStatusEvent(ctx context.Context, msg redis.XMessage) error {
	eventType, ok := msg.Values["event_type"].(string)
	if !ok || eventType != "ServerStatusChanged" {
		return nil
	}

	serverID, ok := msg.Values["server_id"].(string)
	if !ok {
		return nil
	}

	payloadStr, ok := msg.Values["payload"].(string)
	if !ok {
		return nil
	}

	var payload struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		return err
	}

	logger.Log.Info("Received ServerStatusChanged event, updating local DB", zap.String("server_id", serverID), zap.String("status", payload.Status))
	return c.repo.UpdateReportingServerStatus(ctx, serverID, payload.Status)
}
