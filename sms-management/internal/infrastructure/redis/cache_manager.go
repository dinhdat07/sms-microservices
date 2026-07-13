package redis

import "context"

// CacheManager defines the interface for server cache operations.
// Implementations include ServerCache (real Redis) and mocks for testing.
type CacheManager interface {
	Upsert(ctx context.Context, id, ipv4, status string, retryCount int) error
	BatchUpsert(ctx context.Context, items []CacheUpsertItem) error
	Delete(ctx context.Context, id string) error
}
