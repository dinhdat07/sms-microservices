package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"sms-reporting/internal/domain"
	"sms-reporting/internal/repository"
)

type ReportingService interface {
	RequestReport(ctx context.Context, email string, startDate string, endDate string) error
	ExecuteDailyMaintenance(ctx context.Context, startTime time.Time, endTime time.Time) error
}

type reportingServiceImpl struct {
	repo        repository.ReportingRepository
	worker      ReportingWorker
	rawProvider domain.RawUptimeProvider
}

func NewReportingService(repo repository.ReportingRepository, worker ReportingWorker, rawProvider domain.RawUptimeProvider) ReportingService {
	return &reportingServiceImpl{
		repo:        repo,
		worker:      worker,
		rawProvider: rawProvider,
	}
}

func (s *reportingServiceImpl) RequestReport(ctx context.Context, email string, startDate string, endDate string) error {
	// Parse dates
	// Format is YYYY-MM-DD
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return domain.ErrInvalidDateFormat
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return domain.ErrInvalidDateFormat
	}
	// Make sure end is at the end of the day
	end = end.Add(24 * time.Hour).Add(-time.Nanosecond)

	correlationID := uuid.New().String()

	req, err := domain.NewReportRequest(email, start, end, correlationID)
	if err != nil {
		return err
	}

	err = s.repo.CreateReportRequest(ctx, req)
	if err != nil {
		return err
	}

	// Enqueue report request to the worker pool
	s.worker.EnqueueReport(req)

	return nil
}

func (s *reportingServiceImpl) ExecuteDailyMaintenance(ctx context.Context, startTime time.Time, endTime time.Time) error {
	// 1. Rollup Data
	if s.rawProvider != nil {
		successCount, totalCount, err := s.rawProvider.CalculateRawUptimeStats(ctx, startTime, endTime.Add(-time.Nanosecond))
		if err != nil {
			return err
		}

		stat := &domain.DailyUptimeStat{
			ID:               uuid.New(),
			Date:             startTime,
			TotalPingCount:   totalCount,
			SuccessPingCount: successCount,
		}

		if err := s.repo.SaveDailyUptimeStat(ctx, stat); err != nil {
			return err
		}
	}

	// 2. Cleanup Old Data
	if s.rawProvider != nil {
		// Clean up data older than 7 days from now
		olderThan := time.Now().Add(-7 * 24 * time.Hour)
		if err := s.rawProvider.CleanupOldData(ctx, olderThan); err != nil {
			return err
		}
	}

	return nil
}
