package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	infraRedis "sms-monitoring/internal/infrastructure/redis"
	mockService "sms-monitoring/internal/service/mock"
	"sms-monitoring/internal/worker/checkers"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockHealthChecker struct {
	mock.Mock
}

func (m *mockHealthChecker) Check(ctx context.Context, config checkers.ServerConfig) bool {
	args := m.Called(ctx, config)
	return args.Bool(0)
}

type mockFactory struct {
	mock.Mock
}

func (m *mockFactory) GetChecker(method string) checkers.HealthChecker {
	args := m.Called(method)
	return args.Get(0).(checkers.HealthChecker)
}

func TestWorkerPool_Run(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	db, mockRedis := redismock.NewClientMock()
	monService := mockService.NewMonitoringService(t)
	pinger := new(mockFactory)

	pool := NewWorkerPool(db, monService, pinger, 1, 1*time.Second)

	checker := new(mockHealthChecker)
	pinger.On("GetChecker", "ICMP").Return(checker)
	checker.On("Check", mock.Anything, checkers.ServerConfig{"ipv4": "1.1.1.1", "health_check_method": "ICMP", "server_id": "id-1"}).Return(true)

	mockRedis.ExpectBLPop(2*time.Second, infraRedis.MonitoringQueueKey).SetVal([]string{infraRedis.MonitoringQueueKey, "id-1"})
	mockRedis.ExpectHGetAll("server:info:id-1").SetVal(map[string]string{"ipv4": "1.1.1.1", "health_check_method": "ICMP"})

	monService.On("Evaluate", mock.Anything, "id-1", "1.1.1.1", true).Return(nil).Run(func(args mock.Arguments) {
		cancel() // Cancel context to stop the loop after processing one item
	})

	err := pool.Run(ctx)
	assert.NoError(t, err)
	assert.NoError(t, mockRedis.ExpectationsWereMet())
	pinger.AssertExpectations(t)
	checker.AssertExpectations(t)
	monService.AssertExpectations(t)
}

func TestWorkerPool_Run_EmptyQueue(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	db, mockRedis := redismock.NewClientMock()
	pool := NewWorkerPool(db, nil, nil, 1, 1*time.Second)

	mockRedis.ExpectBLPop(2*time.Second, infraRedis.MonitoringQueueKey).SetErr(redis.Nil)

	err := pool.Run(ctx)
	assert.NoError(t, err)
	assert.NoError(t, mockRedis.ExpectationsWereMet())
}

func TestWorkerPool_Run_RedisError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	db, mockRedis := redismock.NewClientMock()
	pool := NewWorkerPool(db, nil, nil, 1, 1*time.Second)

	mockRedis.ExpectBLPop(2*time.Second, infraRedis.MonitoringQueueKey).SetErr(errors.New("redis error"))

	err := pool.Run(ctx)
	assert.NoError(t, err)
	assert.NoError(t, mockRedis.ExpectationsWereMet())
}

func TestProcessServer_HGetError(t *testing.T) {
	db, mockRedis := redismock.NewClientMock()
	monService := mockService.NewMonitoringService(t)
	pinger := new(mockFactory)
	ctx := context.Background()
	pool := NewWorkerPool(db, monService, pinger, 1, 1*time.Second)

	mockRedis.ExpectHGetAll("server:info:srv-1").SetErr(errors.New("redis err"))

	// It should return early, so no calls to Ping or Evaluate expected
	pool.(*workerPool).processServer(ctx, "srv-1")
	assert.NoError(t, mockRedis.ExpectationsWereMet())
}

func TestProcessServer_EmptyIPv4(t *testing.T) {
	db, mockRedis := redismock.NewClientMock()
	monService := mockService.NewMonitoringService(t)
	pinger := new(mockFactory)
	ctx := context.Background()
	pool := NewWorkerPool(db, monService, pinger, 1, 1*time.Second)

	mockRedis.ExpectHGetAll("server:info:srv-1").SetVal(map[string]string{})

	// It should return early
	pool.(*workerPool).processServer(ctx, "srv-1")
	assert.NoError(t, mockRedis.ExpectationsWereMet())
}

func TestProcessServer_EvaluateError(t *testing.T) {
	db, mockRedis := redismock.NewClientMock()
	monService := mockService.NewMonitoringService(t)
	pinger := new(mockFactory)
	ctx := context.Background()
	pool := NewWorkerPool(db, monService, pinger, 1, 1*time.Second)

	mockRedis.ExpectHGetAll("server:info:srv-1").SetVal(map[string]string{"ipv4": "1.1.1.1", "health_check_method": "ICMP"})
	checker := new(mockHealthChecker)
	pinger.On("GetChecker", "ICMP").Return(checker)
	checker.On("Check", mock.Anything, checkers.ServerConfig{"ipv4": "1.1.1.1", "health_check_method": "ICMP", "server_id": "srv-1"}).Return(false)

	monService.On("Evaluate", mock.Anything, "srv-1", "1.1.1.1", false).Return(errors.New("eval err"))

	pool.(*workerPool).processServer(ctx, "srv-1")
	assert.NoError(t, mockRedis.ExpectationsWereMet())
	pinger.AssertExpectations(t)
	checker.AssertExpectations(t)
	monService.AssertExpectations(t)
}
