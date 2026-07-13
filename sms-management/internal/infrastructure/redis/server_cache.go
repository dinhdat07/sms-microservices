package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const (
	ServerAllIDsKey  = "server:all_ids"
	ServerInfoKeyFmt = "server:info:%s"
)

type ServerCache struct {
	client redis.UniversalClient
}

func NewServerCache(client redis.UniversalClient) *ServerCache {
	return &ServerCache{client: client}
}

// Upsert adds/updates the server in the cache
func (c *ServerCache) Upsert(ctx context.Context, id, ipv4, status string, retryCount int) error {
	if c.client == nil {
		return nil // Redis is disabled
	}

	infoKey := fmt.Sprintf(ServerInfoKeyFmt, id)

	// Update Hash
	err := c.client.HSet(ctx, infoKey, map[string]interface{}{
		"ipv4":        ipv4,
		"status":      status,
		"retry_count": retryCount,
	}).Err()
	if err != nil {
		return fmt.Errorf("hset server info: %w", err)
	}

	// Add to Set
	err = c.client.SAdd(ctx, ServerAllIDsKey, id).Err()
	if err != nil {
		return fmt.Errorf("sadd server all_ids: %w", err)
	}

	return nil
}

type CacheUpsertItem struct {
	ID         string
	IPv4       string
	Status     string
	RetryCount int
}

// BatchUpsert adds/updates multiple servers in the cache using a Pipeline
func (c *ServerCache) BatchUpsert(ctx context.Context, items []CacheUpsertItem) error {
	if c.client == nil || len(items) == 0 {
		return nil
	}

	pipeline := c.client.Pipeline()
	for _, item := range items {
		infoKey := fmt.Sprintf(ServerInfoKeyFmt, item.ID)
		pipeline.HSet(ctx, infoKey, map[string]interface{}{
			"ipv4":        item.IPv4,
			"status":      item.Status,
			"retry_count": item.RetryCount,
		})
		pipeline.SAdd(ctx, ServerAllIDsKey, item.ID)
	}

	_, err := pipeline.Exec(ctx)
	if err != nil {
		return fmt.Errorf("redis pipeline batch upsert: %w", err)
	}

	return nil
}

// Delete removes the server from the cache
func (c *ServerCache) Delete(ctx context.Context, id string) error {
	if c.client == nil {
		return nil
	}

	infoKey := fmt.Sprintf(ServerInfoKeyFmt, id)

	// Delete Hash
	err := c.client.Del(ctx, infoKey).Err()
	if err != nil {
		return fmt.Errorf("del server info: %w", err)
	}

	// Remove from Set
	err = c.client.SRem(ctx, ServerAllIDsKey, id).Err()
	if err != nil {
		return fmt.Errorf("srem server all_ids: %w", err)
	}

	return nil
}
