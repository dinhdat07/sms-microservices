package messagebroker

import (
	"context"
	"fmt"

	"sms-management/internal/domain"

	"github.com/redis/go-redis/v9"
)

type RedisPublisher struct {
	client redis.UniversalClient
}

func NewRedisPublisher(client redis.UniversalClient) Publisher {
	return &RedisPublisher{
		client: client,
	}
}

func (p *RedisPublisher) PublishOutboxBatch(ctx context.Context, stream string, events []*domain.OutboxEvent) error {
	if len(events) == 0 {
		return nil
	}

	pipe := p.client.Pipeline()
	for _, event := range events {
		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: stream,
			Values: map[string]interface{}{
				"aggregate_type": event.AggregateType,
				"aggregate_id":   event.AggregateID,
				"event_type":     string(event.EventType),
				"payload":        string(event.Payload),
			},
		})
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute redis pipeline: %w", err)
	}

	return nil
}
