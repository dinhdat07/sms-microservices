package service

import (
	"context"
	"errors"
	"testing"
	"encoding/json"

	monitoringDomain "sms-monitoring/internal/domain"
	mockRepo "sms-monitoring/internal/repository/mock"
	esMock "sms-monitoring/internal/infrastructure/elasticsearch/mock"
	"sms-monitoring/internal/infrastructure/messagebroker"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestEvaluate_FirstFailureStaysOnline(t *testing.T) {
	ctx := context.Background()
	serverID := "svr-123"

	stateStore := mockRepo.NewMockServerStateStore(t)
	db, mockRedis := redismock.NewClientMock()
	esLogger := esMock.NewObservationLogger(t)
	publisher := messagebroker.NewRedisPublisher(db)
	service := NewMonitoringService(publisher, stateStore, esLogger, 2)

	stateStore.On("GetServerState", ctx, serverID).Return(&monitoringDomain.ServerState{Status: "ONLINE", RetryCount: 0}, nil).Once()
	stateStore.On("SetServerState", ctx, serverID, "ONLINE", 1).Return(nil).Once()
	esLogger.On("LogObservation", ctx, serverID, false).Return(nil).Once()

	err := service.Evaluate(ctx, serverID, "1.1.1.1", false)
	assert.NoError(t, err)
	assert.NoError(t, mockRedis.ExpectationsWereMet())
	stateStore.AssertExpectations(t)
}

func TestEvaluate_SecondFailureGoesOffline(t *testing.T) {
	ctx := context.Background()
	serverID := "svr-123"

	stateStore := mockRepo.NewMockServerStateStore(t)
	db, mockRedis := redismock.NewClientMock()
	esLogger := esMock.NewObservationLogger(t)
	publisher := messagebroker.NewRedisPublisher(db)
	service := NewMonitoringService(publisher, stateStore, esLogger, 2)

	stateStore.On("GetServerState", ctx, serverID).Return(&monitoringDomain.ServerState{Status: "ONLINE", RetryCount: 1}, nil).Once()
	stateStore.On("SetServerState", ctx, serverID, "OFFLINE", 0).Return(nil).Once()
	esLogger.On("LogObservation", ctx, serverID, false).Return(nil).Once()
	
	payload, _ := json.Marshal(map[string]interface{}{
		"id": serverID,
		"status": "OFFLINE",
		"retry_count": 0,
	})
	mockRedis.ExpectXAdd(&redis.XAddArgs{
		Stream: "sms.events.server_status",
		Values: []interface{}{
			"server_id", serverID,
			"event_type", "ServerStatusChanged",
			"payload", string(payload),
		},
	}).SetVal("1-0")

	err := service.Evaluate(ctx, serverID, "1.1.1.1", false)
	assert.NoError(t, err)
	assert.NoError(t, mockRedis.ExpectationsWereMet())
	stateStore.AssertExpectations(t)
}

func TestEvaluate_RedisXAddError(t *testing.T) {
	ctx := context.Background()
	serverID := "svr-789"

	stateStore := mockRepo.NewMockServerStateStore(t)
	db, mockRedis := redismock.NewClientMock()
	esLogger := esMock.NewObservationLogger(t)
	publisher := messagebroker.NewRedisPublisher(db)
	service := NewMonitoringService(publisher, stateStore, esLogger, 2)

	stateStore.On("GetServerState", ctx, serverID).Return(&monitoringDomain.ServerState{Status: "OFFLINE", RetryCount: 0}, nil).Once()
	stateStore.On("SetServerState", ctx, serverID, "ONLINE", 0).Return(nil).Once()

	esLogger.On("LogObservation", ctx, serverID, true).Return(nil).Once()
	
	payload, _ := json.Marshal(map[string]interface{}{
		"id": serverID,
		"status": "ONLINE",
		"retry_count": 0,
	})
	mockRedis.ExpectXAdd(&redis.XAddArgs{
		Stream: "sms.events.server_status",
		Values: []interface{}{
			"server_id", serverID,
			"event_type", "ServerStatusChanged",
			"payload", string(payload),
		},
	}).SetErr(errors.New("redis stream error"))

	err := service.Evaluate(ctx, serverID, "1.1.1.1", true)
	assert.NoError(t, err)
	assert.NoError(t, mockRedis.ExpectationsWereMet())
	stateStore.AssertExpectations(t)
}

