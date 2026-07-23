package service

import (
	"context"
	"testing"
	"time"

	"sms-reporting/internal/domain"
	repoMock "sms-reporting/internal/repository/mock"
	domainMock "sms-reporting/internal/domain/mock"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestReportingWorker_Lifecycle(t *testing.T) {
	repo := new(repoMock.MockReportingRepository)
	uptimeCalc := new(domainMock.MockUptimeCalculator)
	notifier := new(domainMock.MockReportNotifier)

	worker := NewReportingWorker(repo, uptimeCalc, 2, 10, notifier)

	ctx := context.Background()
	worker.Start(ctx)

	// Send a valid report request
	req := &domain.ReportRequest{
		ID:        uuid.New(),
		RequestorEmail: "test@example.com",
		StartTime: time.Now().Add(-24 * time.Hour),
		EndTime:   time.Now(),
		Status:    domain.ReportStatusPending,
	}

	repo.On("UpdateReportStatus", ctx, req.ID.String(), domain.ReportStatusProcessing).Return(nil)
	repo.On("GetServerCountByStatus", ctx, "").Return(int64(100), nil)
	repo.On("GetServerCountByStatus", ctx, "ONLINE").Return(int64(80), nil)
	repo.On("GetServerCountByStatus", ctx, "OFFLINE").Return(int64(20), nil)
	repo.On("GetServerCountByStatus", ctx, "UNKNOWN").Return(int64(0), nil)
	
	uptimeCalc.On("CalculateUptime", ctx, req.StartTime, req.EndTime).Return(99.5, nil)
	
	notifier.On("SendReportEmail", ctx, "test@example.com", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateReportStatus", ctx, req.ID.String(), domain.ReportStatusCompleted).Return(nil)

	worker.EnqueueReport(req)

	// Stop gracefully, will process the enqueued report and wait for it
	worker.Stop()

	repo.AssertExpectations(t)
	uptimeCalc.AssertExpectations(t)
	notifier.AssertExpectations(t)
}

func TestReportingWorker_ProcessReport_StatusUpdateFail(t *testing.T) {
	repo := new(repoMock.MockReportingRepository)
	uptimeCalc := new(domainMock.MockUptimeCalculator)
	notifier := new(domainMock.MockReportNotifier)

	worker := NewReportingWorker(repo, uptimeCalc, 1, 10, notifier)
	worker.Start(context.Background())

	req := &domain.ReportRequest{ID: uuid.New()}
	
	repo.On("UpdateReportStatus", mock.Anything, req.ID.String(), domain.ReportStatusProcessing).Return(assert.AnError)

	worker.EnqueueReport(req)
	worker.Stop()

	repo.AssertExpectations(t)
}

func TestPercentage(t *testing.T) {
	assert.Equal(t, 0.0, percentage(5, 0))
	assert.Equal(t, 50.0, percentage(50, 100))
	assert.Equal(t, 33.333333333333336, percentage(1, 3))
}

func TestHealthLabel(t *testing.T) {
	assert.Equal(t, "Healthy", healthLabel(100.0))
	assert.Equal(t, "Healthy", healthLabel(99.0))
	assert.Equal(t, "Attention Recommended", healthLabel(98.9))
	assert.Equal(t, "Attention Recommended", healthLabel(95.0))
	assert.Equal(t, "Action Required", healthLabel(94.9))
	assert.Equal(t, "Action Required", healthLabel(0.0))
}

func TestReportingWorker_DoWork_GetServerCountOnlineFail(t *testing.T) {
	repo := new(repoMock.MockReportingRepository)
	uptimeCalc := new(domainMock.MockUptimeCalculator)
	notifier := new(domainMock.MockReportNotifier)

	worker := NewReportingWorker(repo, uptimeCalc, 1, 10, notifier)
	worker.Start(context.Background())

	req := &domain.ReportRequest{ID: uuid.New()}
	
	repo.On("UpdateReportStatus", mock.Anything, req.ID.String(), domain.ReportStatusProcessing).Return(nil)
	repo.On("GetServerCountByStatus", mock.Anything, "").Return(int64(100), nil)
	repo.On("GetServerCountByStatus", mock.Anything, "ONLINE").Return(int64(0), assert.AnError)
	repo.On("UpdateReportStatus", mock.Anything, req.ID.String(), domain.ReportStatusFailed).Return(nil)

	worker.EnqueueReport(req)
	worker.Stop()

	repo.AssertExpectations(t)
}

func TestReportingWorker_DoWork_GetServerCountOfflineFail(t *testing.T) {
	repo := new(repoMock.MockReportingRepository)
	uptimeCalc := new(domainMock.MockUptimeCalculator)
	notifier := new(domainMock.MockReportNotifier)

	worker := NewReportingWorker(repo, uptimeCalc, 1, 10, notifier)
	worker.Start(context.Background())

	req := &domain.ReportRequest{ID: uuid.New()}
	
	repo.On("UpdateReportStatus", mock.Anything, req.ID.String(), domain.ReportStatusProcessing).Return(nil)
	repo.On("GetServerCountByStatus", mock.Anything, "").Return(int64(100), nil)
	repo.On("GetServerCountByStatus", mock.Anything, "ONLINE").Return(int64(80), nil)
	repo.On("GetServerCountByStatus", mock.Anything, "OFFLINE").Return(int64(0), assert.AnError)
	repo.On("UpdateReportStatus", mock.Anything, req.ID.String(), domain.ReportStatusFailed).Return(nil)

	worker.EnqueueReport(req)
	worker.Stop()

	repo.AssertExpectations(t)
}

func TestNewReportingWorker_Defaults(t *testing.T) {
	worker := NewReportingWorker(nil, nil, 0, 0, nil).(*reportingWorkerImpl)
	assert.Equal(t, 5, worker.workerCount)
	assert.Equal(t, 100, cap(worker.jobQueue))
}

func TestReportingWorker_DoWork_ESError(t *testing.T) {
	repo := new(repoMock.MockReportingRepository)
	uptimeCalc := new(domainMock.MockUptimeCalculator)
	notifier := new(domainMock.MockReportNotifier)

	worker := NewReportingWorker(repo, uptimeCalc, 1, 10, notifier)
	worker.Start(context.Background())

	req := &domain.ReportRequest{ID: uuid.New()}
	
	repo.On("UpdateReportStatus", mock.Anything, req.ID.String(), domain.ReportStatusProcessing).Return(nil)
	repo.On("GetServerCountByStatus", mock.Anything, "").Return(int64(100), nil)
	repo.On("GetServerCountByStatus", mock.Anything, "ONLINE").Return(int64(80), nil)
	repo.On("GetServerCountByStatus", mock.Anything, "OFFLINE").Return(int64(20), nil)
	repo.On("GetServerCountByStatus", mock.Anything, "UNKNOWN").Return(int64(0), nil)
	
	uptimeCalc.On("CalculateUptime", mock.Anything, mock.Anything, mock.Anything).Return(float64(0), assert.AnError)
	
	repo.On("UpdateReportStatus", mock.Anything, req.ID.String(), domain.ReportStatusFailed).Return(nil)

	worker.EnqueueReport(req)
	worker.Stop()

	repo.AssertExpectations(t)
}

func TestReportingWorker_DoWork_NotifierError(t *testing.T) {
	repo := new(repoMock.MockReportingRepository)
	uptimeCalc := new(domainMock.MockUptimeCalculator)
	notifier := new(domainMock.MockReportNotifier)

	worker := NewReportingWorker(repo, uptimeCalc, 1, 10, notifier)
	worker.Start(context.Background())

	req := &domain.ReportRequest{ID: uuid.New()}
	
	repo.On("UpdateReportStatus", mock.Anything, req.ID.String(), domain.ReportStatusProcessing).Return(nil)
	repo.On("GetServerCountByStatus", mock.Anything, "").Return(int64(100), nil)
	repo.On("GetServerCountByStatus", mock.Anything, "ONLINE").Return(int64(80), nil)
	repo.On("GetServerCountByStatus", mock.Anything, "OFFLINE").Return(int64(20), nil)
	repo.On("GetServerCountByStatus", mock.Anything, "UNKNOWN").Return(int64(0), nil)
	
	uptimeCalc.On("CalculateUptime", mock.Anything, mock.Anything, mock.Anything).Return(float64(100), nil)
	
	notifier.On("SendReportEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)
	
	repo.On("UpdateReportStatus", mock.Anything, req.ID.String(), domain.ReportStatusFailed).Return(nil)

	worker.EnqueueReport(req)
	worker.Stop()

	repo.AssertExpectations(t)
}

func TestReportingWorker_DoWork_GetServerCountTotalFail(t *testing.T) {
	repo := new(repoMock.MockReportingRepository)
	uptimeCalc := new(domainMock.MockUptimeCalculator)
	notifier := new(domainMock.MockReportNotifier)

	worker := NewReportingWorker(repo, uptimeCalc, 1, 10, notifier)
	worker.Start(context.Background())

	req := &domain.ReportRequest{ID: uuid.New()}
	
	repo.On("UpdateReportStatus", mock.Anything, req.ID.String(), domain.ReportStatusProcessing).Return(nil)
	repo.On("GetServerCountByStatus", mock.Anything, "").Return(int64(0), assert.AnError)
	repo.On("UpdateReportStatus", mock.Anything, req.ID.String(), domain.ReportStatusFailed).Return(nil)

	worker.EnqueueReport(req)
	worker.Stop()

	repo.AssertExpectations(t)
}

func TestReportingWorker_ProcessReport_FinalStatusUpdateFail(t *testing.T) {
	repo := new(repoMock.MockReportingRepository)
	uptimeCalc := new(domainMock.MockUptimeCalculator)
	notifier := new(domainMock.MockReportNotifier)

	worker := NewReportingWorker(repo, uptimeCalc, 1, 10, notifier)
	worker.Start(context.Background())

	req := &domain.ReportRequest{ID: uuid.New()}
	
	repo.On("UpdateReportStatus", mock.Anything, req.ID.String(), domain.ReportStatusProcessing).Return(nil)
	repo.On("GetServerCountByStatus", mock.Anything, "").Return(int64(0), assert.AnError)
	// Fail the final status update
	repo.On("UpdateReportStatus", mock.Anything, req.ID.String(), domain.ReportStatusFailed).Return(assert.AnError)

	worker.EnqueueReport(req)
	worker.Stop()

	repo.AssertExpectations(t)
}
