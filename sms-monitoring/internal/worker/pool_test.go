package worker

import (
	"context"
	"testing"
	"time"
	"errors"

	

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	mockService "sms-monitoring/internal/service/mock"
)

type mockPinger struct {
	mock.Mock
}

func (m *mockPinger) Ping(ip string, timeout time.Duration) bool {
	args := m.Called(ip, timeout)
	return args.Bool(0)
}

type mockMonitoringService struct {
	mock.Mock
}

func (m *mockMonitoringService) Evaluate(ctx context.Context, serverID string, ip string, pingSuccess bool) error {
	args := m.Called(ctx, serverID, ip, pingSuccess)
	return args.Error(0)
}

func TestWorkerPool_Run(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	db, mockRedis := redismock.NewClientMock()
	monService := mockService.NewMonitoringService(t)
	pinger := new(mockPinger)

	pool := NewWorkerPool(db, monService, pinger, 1, 1*time.Second)

	mockRedis.ExpectBLPop(2*time.Second, "monitoring:queue").SetVal([]string{"monitoring:queue", "id-1"})
	mockRedis.ExpectHGet("server:info:id-1", "ipv4").SetVal("1.1.1.1")
	pinger.On("Ping", "1.1.1.1", 1*time.Second).Return(true)
	monService.On("Evaluate", mock.Anything, "id-1", "1.1.1.1", true).Return(nil).Run(func(args mock.Arguments) {
		cancel() // Cancel context to stop the loop after processing one item
	})

	err := pool.Run(ctx)
	assert.NoError(t, err)
	assert.NoError(t, mockRedis.ExpectationsWereMet())
	pinger.AssertExpectations(t)
	monService.AssertExpectations(t)
}

func TestWorkerPool_Run_EmptyQueue(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	db, mockRedis := redismock.NewClientMock()
	pool := NewWorkerPool(db, nil, nil, 1, 1*time.Second)

	mockRedis.ExpectBLPop(2*time.Second, "monitoring:queue").SetErr(redis.Nil)

	err := pool.Run(ctx)
	assert.NoError(t, err)
	assert.NoError(t, mockRedis.ExpectationsWereMet())
}

func TestWorkerPool_Run_RedisError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	db, mockRedis := redismock.NewClientMock()
	pool := NewWorkerPool(db, nil, nil, 1, 1*time.Second)

	mockRedis.ExpectBLPop(2*time.Second, "monitoring:queue").SetErr(errors.New("redis error"))

	err := pool.Run(ctx)
	assert.NoError(t, err)
	assert.NoError(t, mockRedis.ExpectationsWereMet())
}


func TestProcessServer_HGetError(t *testing.T) {
	db, mockRedis := redismock.NewClientMock()
	monService := mockService.NewMonitoringService(t)
	pinger := new(mockPinger)
	ctx := context.Background()
	pool := NewWorkerPool(db, monService, pinger, 1, 1*time.Second)

	mockRedis.ExpectHGet("server:info:srv-1", "ipv4").SetErr(errors.New("redis err"))

	// It should return early, so no calls to Ping or Evaluate expected
	pool.(*workerPool).processServer(ctx, "srv-1")
	assert.NoError(t, mockRedis.ExpectationsWereMet())
}

func TestProcessServer_EmptyIPv4(t *testing.T) {
	db, mockRedis := redismock.NewClientMock()
	monService := mockService.NewMonitoringService(t)
	pinger := new(mockPinger)
	ctx := context.Background()
	pool := NewWorkerPool(db, monService, pinger, 1, 1*time.Second)

	mockRedis.ExpectHGet("server:info:srv-1", "ipv4").SetVal("")

	// It should return early
	pool.(*workerPool).processServer(ctx, "srv-1")
	assert.NoError(t, mockRedis.ExpectationsWereMet())
}

func TestProcessServer_EvaluateError(t *testing.T) {
	db, mockRedis := redismock.NewClientMock()
	monService := mockService.NewMonitoringService(t)
	pinger := new(mockPinger)
	ctx := context.Background()
	pool := NewWorkerPool(db, monService, pinger, 1, 1*time.Second)

	mockRedis.ExpectHGet("server:info:srv-1", "ipv4").SetVal("10.0.0.1")
	pinger.On("Ping", "10.0.0.1", 1*time.Second).Return(true)
	monService.On("Evaluate", ctx, "srv-1", "10.0.0.1", true).Return(errors.New("eval err"))

	pool.(*workerPool).processServer(ctx, "srv-1")
	assert.NoError(t, mockRedis.ExpectationsWereMet())
	pinger.AssertExpectations(t)
	monService.AssertExpectations(t)
}

