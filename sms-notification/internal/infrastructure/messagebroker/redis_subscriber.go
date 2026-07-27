package messagebroker

import (
	"context"
	"time"

	"sms-notification/internal/infrastructure/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisSubscriber struct {
	client redis.UniversalClient
}

func NewRedisSubscriber(client redis.UniversalClient) Subscriber {
	return &RedisSubscriber{
		client: client,
	}
}

func (s *RedisSubscriber) Subscribe(ctx context.Context, stream string, group string, consumer string, handler MessageHandler) error {
	// Create consumer group if not exists
	err := s.client.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil {
		if err.Error() == "BUSYGROUP Consumer Group name already exists" {
			// ignore
		} else {
			logger.Log.Error("Failed to create consumer group", zap.String("stream", stream), zap.Error(err))
		}
	}

	logger.Log.Info("Starting subscriber loop", zap.String("stream", stream), zap.String("group", group))

	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("Stopping subscriber", zap.String("stream", stream))
			return nil
		default:
			res, err := s.client.XReadGroup(ctx, &redis.XReadGroupArgs{
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
					brokerMsg := Message{
						ID:     msg.ID,
						Values: msg.Values,
					}
					
					// Handle message
					err := handler(ctx, brokerMsg)
					if err != nil {
						logger.Log.Error("Failed to handle message", zap.String("id", msg.ID), zap.Error(err))
					} else {
						// Ack message on success
						s.client.XAck(ctx, stream, group, msg.ID)
					}
				}
			}
		}
	}
}
