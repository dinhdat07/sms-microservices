package service

import (
	"context"
	"encoding/json"

	serverDomain "sms-monitoring/internal/domain"
	"sms-monitoring/internal/infrastructure/elasticsearch"
	"sms-monitoring/internal/infrastructure/logger"
	"sms-monitoring/internal/infrastructure/messagebroker"
	"sms-monitoring/internal/repository"

	"go.uber.org/zap"
)

type MonitoringService interface {
	Evaluate(ctx context.Context, serverID string, ip string, pingSuccess bool) error
}

type monitoringServiceImpl struct {
	publisher        messagebroker.Publisher
	stateStore       repository.ServerStateStore
	esLogger         elasticsearch.ObservationLogger
	failureThreshold int
}

func NewMonitoringService(publisher messagebroker.Publisher, stateStore repository.ServerStateStore, esLogger elasticsearch.ObservationLogger, failureThreshold int) MonitoringService {
	return &monitoringServiceImpl{
		publisher:        publisher,
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
		currentStatusStr = string(serverDomain.ServerStatusUnknown) // Default
	}
	currentStatus := serverDomain.ServerStatus(currentStatusStr)

	retryCount := state.RetryCount

	// State Machine Evaluation
	newStatus := currentStatus
	var statusChanged bool

	if pingSuccess {
		if currentStatus == serverDomain.ServerStatusOffline || currentStatus == serverDomain.ServerStatusUnknown {
			// Recovery Threshold = 1
			newStatus = serverDomain.ServerStatusOnline
			statusChanged = true
			retryCount = 0
		} else {
			// Already online, reset retry count if > 0
			retryCount = 0
		}
	} else {
		if currentStatus == serverDomain.ServerStatusOnline || currentStatus == serverDomain.ServerStatusUnknown {
			retryCount++
			if retryCount >= s.failureThreshold || currentStatus == serverDomain.ServerStatusUnknown {
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
	err = s.stateStore.SetServerState(ctx, serverID, string(newStatus), retryCount)
	if err != nil {
		return err
	}

	// Publish Event via Publisher ONLY if state actually changes
	if statusChanged {
		payload, _ := json.Marshal(map[string]interface{}{
			"server_id":   serverID,
			"status":      newStatus,
			"retry_count": retryCount,
		})

		err = s.publisher.Publish(ctx, "sms.events.server_status", []interface{}{
			"server_id", serverID,
			"event_type", "ServerStatusChanged",
			"payload", string(payload),
		})
		if err != nil {
			logger.Log.Error("Failed to publish server status event", zap.Error(err))
		}
	}

	return nil
}
