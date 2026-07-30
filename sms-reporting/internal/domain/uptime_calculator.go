package domain

import (
	"context"
	"time"
)

// UptimeCalculator is the high-level business interface for calculating uptime percentage.
type UptimeCalculator interface {
	CalculateUptime(ctx context.Context, startTime time.Time, endTime time.Time) (float64, error)
}

// RawUptimeProvider provides raw ping counts and maintenance operations for the underlying storage.
type RawUptimeProvider interface {
	CalculateRawUptimeStats(ctx context.Context, startTime time.Time, endTime time.Time) (int64, int64, error)
	CleanupOldData(ctx context.Context, olderThan time.Time) error
}
