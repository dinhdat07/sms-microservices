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
}

type reportingServiceImpl struct {
	repo   repository.ReportingRepository
	worker ReportingWorker
}

func NewReportingService(repo repository.ReportingRepository, worker ReportingWorker) ReportingService {
	return &reportingServiceImpl{
		repo:   repo,
		worker: worker,
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
