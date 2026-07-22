package app

import (
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	mockWorker "sms-reporting/internal/service/mock"
)

func TestScheduler_StartStop(t *testing.T) {
	db, _ := redismock.NewClientMock()
	worker := mockWorker.NewMockReportingWorker(t)
	
	scheduler := NewScheduler(worker, db, "* * * * *", "")
	assert.NotNil(t, scheduler)
	
	err := scheduler.Start()
	assert.NoError(t, err)
	
	scheduler.Stop()
}

func TestScheduler_JobExecution_LockAcquired(t *testing.T) {
	db, mockRedis := redismock.NewClientMock()
	worker := mockWorker.NewMockReportingWorker(t)
	
	scheduler := NewScheduler(worker, db, "* * * * *", "test@example.com")
	
	mockRedis.ExpectSetNX("lock:reporting_scheduler", "locked", 30*time.Second).SetVal(true)
	
	worker.On("EnqueueReport", mock.AnythingOfType("*domain.ReportRequest")).Return()
	
	err := scheduler.Start()
	assert.NoError(t, err)
	
	// Manually invoke the job
	scheduler.cron.Entries()[0].Job.Run()
	
	scheduler.Stop()
	
	assert.NoError(t, mockRedis.ExpectationsWereMet())
}

func TestScheduler_JobExecution_LockNotAcquired(t *testing.T) {
	db, mockRedis := redismock.NewClientMock()
	worker := mockWorker.NewMockReportingWorker(t)
	
	scheduler := NewScheduler(worker, db, "* * * * *", "test@example.com")
	
	mockRedis.ExpectSetNX("lock:reporting_scheduler", "locked", 30*time.Second).SetVal(false)
	
	err := scheduler.Start()
	assert.NoError(t, err)
	
	// Manually invoke the job
	scheduler.cron.Entries()[0].Job.Run()
	
	scheduler.Stop()
	
	assert.NoError(t, mockRedis.ExpectationsWereMet())
	worker.AssertNotCalled(t, "EnqueueReport", mock.Anything)
}
