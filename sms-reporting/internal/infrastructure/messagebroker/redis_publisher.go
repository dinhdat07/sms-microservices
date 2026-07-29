package messagebroker

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisPublisher struct {
	client redis.UniversalClient
	maxLen int64
}

func NewRedisPublisher(client redis.UniversalClient, maxLen int64) Publisher {
	return &RedisPublisher{
		client: client,
		maxLen: maxLen,
	}
}

func (p *RedisPublisher) Publish(ctx context.Context, stream string, values []interface{}) error {
	return p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		MaxLen: p.maxLen,
		Approx: true,
		Values: values,
	}).Err()
}
