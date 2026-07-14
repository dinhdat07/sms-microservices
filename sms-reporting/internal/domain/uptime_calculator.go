package domain

import (
	"context"
	"time"
)

// UptimeCalculator computes uptime percentage from observation logs.
type UptimeCalculator interface {
	CalculateUptime(ctx context.Context, startTime, endTime time.Time) (float64, error)
}
