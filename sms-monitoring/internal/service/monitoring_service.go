package service

import (
	"context"

	"encoding/json"

	serverDomain "sms-monitoring/internal/domain"
	"sms-monitoring/internal/infrastructure/elasticsearch"
	"sms-monitoring/internal/repository"

	"github.com/redis/go-redis/v9"
)

type MonitoringService interface {
	Evaluate(ctx context.Context, serverID string, ip string, pingSuccess bool) error
}

type monitoringServiceImpl struct {
	rdb              redis.UniversalClient
	stateStore       repository.ServerStateStore
	esLogger         elasticsearch.ObservationLogger
	failureThreshold int
}

func NewMonitoringService(rdb redis.UniversalClient, stateStore repository.ServerStateStore, esLogger elasticsearch.ObservationLogger, failureThreshold int) MonitoringService {
	return &monitoringServiceImpl{
		rdb:              rdb,
		stateStore:       stateStore,
		esLogger:         esLogger,
		failureThreshold: failureThreshold,
	}
}

func (s *monitoringServiceImpl) Evaluate(ctx context.Context, serverID string, ip string, pingSuccess bool) error {
	// Fire-and-forget: buffered, non-blocking, flushed in bulk
	s.esLogger.LogObservation(ctx, serverID, pingSuccess)

	// Fetch current status and retry count from state store (Redis)
	state, err := s.stateStore.GetServerState(ctx, serverID)
	if err != nil {
		return err
	}

	currentStatusStr := state.Status
	if currentStatusStr == "" {
		currentStatusStr = string(serverDomain.ServerStatusOnline) // Default
	}
	currentStatus := serverDomain.ServerStatus(currentStatusStr)

	retryCount := state.RetryCount

	// State Machine Evaluation
	var newStatus serverDomain.ServerStatus
	var statusChanged bool

	if pingSuccess {
		if currentStatus == serverDomain.ServerStatusOffline {
			// Recovery Threshold = 1
			newStatus = serverDomain.ServerStatusOnline
			statusChanged = true
			retryCount = 0
		} else {
			// Already online, reset retry count if > 0
			retryCount = 0
		}
	} else {
		if currentStatus == serverDomain.ServerStatusOnline {
			retryCount++
			if retryCount >= s.failureThreshold {
				newStatus = serverDomain.ServerStatusOffline
				statusChanged = true
				retryCount = 0
			}
		} else {
			// Already offline
			retryCount++
		}
	}

	// Update Redis cache
	if statusChanged {
		err = s.stateStore.SetServerState(ctx, serverID, string(newStatus), retryCount)
	} else {
		err = s.stateStore.SetServerState(ctx, serverID, string(currentStatus), retryCount)
	}
	if err != nil {
		return err
	}

	// Publish Event to Redis Stream ONLY if state actually changes
	if statusChanged {
		payload, _ := json.Marshal(map[string]interface{}{
			"id":          serverID,
			"status":      newStatus,
			"retry_count": retryCount,
		})

		// use []interface instead of map[string] for fixed order
		s.rdb.XAdd(ctx, &redis.XAddArgs{
			Stream: "sms.events.server_status",
			Values: []interface{}{
				"server_id", serverID,
				"event_type", "ServerStatusChanged",
				"payload", string(payload),
			},
		})
	}

	return nil
}
