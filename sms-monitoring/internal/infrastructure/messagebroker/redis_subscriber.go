package messagebroker

import (
	"context"
	"time"

	"sms-monitoring/internal/infrastructure/logger"

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

	// Self-Recovery: process messages currently in this consumer's PEL
	s.recoverPendingMessages(ctx, stream, group, consumer, handler)

	// Auto-Claim worker: run in background to reclaim stale messages from dead consumers
	go s.autoClaimLoop(ctx, stream, group, consumer, handler)

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

func (s *RedisSubscriber) recoverPendingMessages(ctx context.Context, stream string, group string, consumer string, handler MessageHandler) {
	logger.Log.Info("Recovering pending messages (PEL) for consumer", zap.String("stream", stream), zap.String("consumer", consumer))
	for {
		res, err := s.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    group,
			Consumer: consumer,
			Streams:  []string{stream, "0"},
			Count:    50,
			Block:    0,
		}).Result()
		
		if err != nil && err != redis.Nil {
			logger.Log.Error("Failed to recover pending messages", zap.Error(err))
			break
		}

		if len(res) == 0 || len(res[0].Messages) == 0 {
			break
		}

		for _, msg := range res[0].Messages {
			brokerMsg := Message{ID: msg.ID, Values: msg.Values}
			if err := handler(ctx, brokerMsg); err != nil {
				logger.Log.Error("Failed to handle pending message", zap.String("id", msg.ID), zap.Error(err))
			} else {
				s.client.XAck(ctx, stream, group, msg.ID)
			}
		}

		if len(res[0].Messages) < 50 {
			break
		}
	}
}

func (s *RedisSubscriber) autoClaimLoop(ctx context.Context, stream string, group string, consumer string, handler MessageHandler) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	minIdle := 5 * time.Minute

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.claimAndProcessStaleMessages(ctx, stream, group, consumer, minIdle, handler)
		}
	}
}

func (s *RedisSubscriber) claimAndProcessStaleMessages(ctx context.Context, stream, group, consumer string, minIdle time.Duration, handler MessageHandler) {
	start := "-"
	dlqKey := stream + ":dlq_counters"

	for {
		messages, nextStart, err := s.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream:   stream,
			Group:    group,
			MinIdle:  minIdle,
			Start:    start,
			Count:    50,
			Consumer: consumer,
		}).Result()

		if err != nil && err != redis.Nil {
			logger.Log.Error("Error claiming stale messages", zap.Error(err))
			break
		}

		for _, msg := range messages {
			// DLQ Logic: Track retries manually
			retries, _ := s.client.HIncrBy(ctx, dlqKey, msg.ID, 1).Result()
			if retries >= 3 {
				logger.Log.Warn("Message exceeded retry limit, moving to DLQ", zap.String("id", msg.ID))
				s.client.XAdd(ctx, &redis.XAddArgs{
					Stream: stream + ":dlq",
					Values: msg.Values,
				})
				s.client.XAck(ctx, stream, group, msg.ID)
				s.client.HDel(ctx, dlqKey, msg.ID)
				continue
			}

			brokerMsg := Message{ID: msg.ID, Values: msg.Values}
			if err := handler(ctx, brokerMsg); err != nil {
				logger.Log.Error("Failed to handle claimed message", zap.String("id", msg.ID), zap.Error(err))
			} else {
				s.client.XAck(ctx, stream, group, msg.ID)
				s.client.HDel(ctx, dlqKey, msg.ID)
			}
		}

		if nextStart == "0-0" || nextStart == "" {
			break
		}
		start = nextStart
	}
}
