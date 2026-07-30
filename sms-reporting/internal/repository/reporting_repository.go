package repository

import (
	"context"
	"errors"
	"time"

	"sms-reporting/internal/domain"
)

var ErrRecordNotFound = errors.New("record not found")

type ReportingRepository interface {
	// Report Request operations
	CreateReportRequest(ctx context.Context, req *domain.ReportRequest) error
	UpdateReportStatus(ctx context.Context, id string, status string) error

	// Server stats (query from local reporting_servers table)
	GetServerCountByStatus(ctx context.Context, status string) (int64, error)

	// Data Replication operations (populated via Events)
	UpsertReportingServer(ctx context.Context, server *domain.ReportingServer) error
	UpdateReportingServerStatus(ctx context.Context, serverID string, status string) error
	DeleteReportingServer(ctx context.Context, serverID string) error

	SaveDailyUptimeStat(ctx context.Context, stat *domain.DailyUptimeStat) error
	GetDailyUptimeStats(ctx context.Context, startDate time.Time, endDate time.Time) ([]domain.DailyUptimeStat, error)
}
