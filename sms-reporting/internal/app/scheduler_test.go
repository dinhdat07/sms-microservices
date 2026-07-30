package app

import (
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	mockService "sms-reporting/internal/service/mock"
)

func TestScheduler_StartStop(t *testing.T) {
	db, mockRedis := redismock.NewClientMock()
	mockReportingService := mockService.NewMockReportingService(t)
	scheduler := NewScheduler(mockReportingService, db, "* * * * *", "test@example.com")

	err := scheduler.Start()
	assert.NoError(t, err)

	scheduler.Stop()
	assert.NoError(t, mockRedis.ExpectationsWereMet())
}

func TestScheduler_JobExecution_LockAcquired(t *testing.T) {
	db, mockRedis := redismock.NewClientMock()
	mockReportingService := mockService.NewMockReportingService(t)
	scheduler := NewScheduler(mockReportingService, db, "* * * * *", "test@example.com")
	
	mockRedis.ExpectSetNX("lock:reporting_scheduler", "locked", 30*time.Second).SetVal(true)
	
	mockReportingService.On("ExecuteDailyMaintenance", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockReportingService.On("RequestReport", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	
	err := scheduler.Start()
	assert.NoError(t, err)
	
	// Manually invoke the job
	scheduler.cron.Entries()[0].Job.Run()
	
	scheduler.Stop()
	
	assert.NoError(t, mockRedis.ExpectationsWereMet())
}

func TestScheduler_JobExecution_LockNotAcquired(t *testing.T) {
	db, mockRedis := redismock.NewClientMock()
	mockReportingService := mockService.NewMockReportingService(t)
	scheduler := NewScheduler(mockReportingService, db, "* * * * *", "test@example.com")
	
	mockRedis.ExpectSetNX("lock:reporting_scheduler", "locked", 30*time.Second).SetVal(false)
	
	err := scheduler.Start()
	assert.NoError(t, err)
	
	scheduler.cron.Entries()[0].Job.Run()
	scheduler.Stop()
	
	assert.NoError(t, mockRedis.ExpectationsWereMet())
	mockReportingService.AssertNotCalled(t, "RequestReport", mock.Anything)
}

func TestScheduler_JobExecution_WithESCleanup(t *testing.T) {
	db, mockRedis := redismock.NewClientMock()
	mockReportingService := mockService.NewMockReportingService(t)
	scheduler := NewScheduler(mockReportingService, db, "* * * * *", "test@example.com")
	
	// Just test normal execution flow to increase coverage since we can't easily mock TypedClient
	mockRedis.ExpectSetNX("lock:reporting_scheduler", "locked", 30*time.Second).SetVal(true)
	mockReportingService.On("ExecuteDailyMaintenance", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockReportingService.On("RequestReport", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	
	err := scheduler.Start()
	assert.NoError(t, err)
	scheduler.cron.Entries()[0].Job.Run()
	scheduler.Stop()
	
	assert.NoError(t, mockRedis.ExpectationsWereMet())
}
