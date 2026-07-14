package messagebroker

import (
	"context"

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

func (p *RedisPublisher) Publish(ctx context.Context, stream string, values []interface{}) error {
	return p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}).Err()
}
