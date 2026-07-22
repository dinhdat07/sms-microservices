package service

import (
	"context"
	"testing"

	repoMock "sms-reporting/internal/repository/mock"
	workerMock "sms-reporting/internal/service/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestReportingService_RequestReport(t *testing.T) {
	repo := new(repoMock.MockReportingRepository)
	worker := new(workerMock.MockReportingWorker)

	svc := NewReportingService(repo, worker)

	ctx := context.Background()

	repo.On("CreateReportRequest", ctx, mock.AnythingOfType("*domain.ReportRequest")).Return(nil)
	worker.On("EnqueueReport", mock.AnythingOfType("*domain.ReportRequest")).Return()

	err := svc.RequestReport(ctx, "test@example.com", "2026-07-01", "2026-07-15")
	assert.NoError(t, err)

	repo.AssertExpectations(t)
	worker.AssertExpectations(t)
}

func TestReportingService_RequestReport_InvalidStartDate(t *testing.T) {
	svc := NewReportingService(nil, nil)
	err := svc.RequestReport(context.Background(), "test@example.com", "invalid-date", "2026-07-15")
	assert.Error(t, err)
}

func TestReportingService_RequestReport_InvalidEndDate(t *testing.T) {
	svc := NewReportingService(nil, nil)
	err := svc.RequestReport(context.Background(), "test@example.com", "2026-07-01", "invalid-date")
	assert.Error(t, err)
}

func TestReportingService_RequestReport_InvalidEmail(t *testing.T) {
	repo := new(repoMock.MockReportingRepository)
	worker := new(workerMock.MockReportingWorker)
	svc := NewReportingService(repo, worker)
	err := svc.RequestReport(context.Background(), "", "2026-07-01", "2026-07-15")
	assert.Error(t, err)
}

func TestReportingService_RequestReport_RepoError(t *testing.T) {
	repo := new(repoMock.MockReportingRepository)
	worker := new(workerMock.MockReportingWorker)

	svc := NewReportingService(repo, worker)
	ctx := context.Background()

	repo.On("CreateReportRequest", ctx, mock.AnythingOfType("*domain.ReportRequest")).Return(assert.AnError)

	err := svc.RequestReport(ctx, "test@example.com", "2026-07-01", "2026-07-15")
	assert.Error(t, err)

	repo.AssertExpectations(t)
}
