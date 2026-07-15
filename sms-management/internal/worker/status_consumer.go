package worker

import (
	"context"
	"encoding/json"

	"sms-management/internal/domain"
	"sms-management/internal/infrastructure/logger"
	"sms-management/internal/infrastructure/messagebroker"
	"sms-management/internal/repository"

	"go.uber.org/zap"
)

// StatusConsumer listens for ServerStatusChanged events from sms-monitoring
// and updates the server status in Postgres accordingly.
type StatusConsumer struct {
	subscriber messagebroker.Subscriber
	serverRepo repository.ServerRepository
	stream     string
	group      string
	consumer   string
}

func NewStatusConsumer(subscriber messagebroker.Subscriber, serverRepo repository.ServerRepository) *StatusConsumer {
	return &StatusConsumer{
		subscriber: subscriber,
		serverRepo: serverRepo,
		stream:     "sms.events.server_status",
		group:      "management_group",
		consumer:   "management_worker_1",
	}
}

func (c *StatusConsumer) Start(ctx context.Context) {
	logger.Log.Info("Starting Status Consumer", zap.String("stream", c.stream))

	go func() {
		err := c.subscriber.Subscribe(ctx, c.stream, c.group, c.consumer, c.processMessage)
		if err != nil {
			logger.Log.Error("Status consumer failed", zap.Error(err))
		}
	}()
}

func (c *StatusConsumer) processMessage(ctx context.Context, msg messagebroker.Message) error {
	eventType, ok := msg.Values["event_type"].(string)
	if !ok {
		return nil
	}

	if eventType != "ServerStatusChanged" {
		return nil
	}

	payloadStr, ok := msg.Values["payload"].(string)
	if !ok {
		return nil
	}

	var payload struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}

	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		logger.Log.Error("Failed to unmarshal status event payload", zap.Error(err))
		return err
	}

	if payload.ID == "" || payload.Status == "" {
		return nil
	}

	// Update server status in Postgres
	server, err := c.serverRepo.GetByID(ctx, payload.ID)
	if err != nil {
		logger.Log.Sugar().Debugf("Server %s not found in DB, skipping status update", payload.ID)
		return nil // Don't retry for not-found servers
	}

	newStatus := domain.ServerStatus(payload.Status)
	if server.CurrentStatus == newStatus {
		return nil // No change needed
	}

	server.CurrentStatus = newStatus
	if err := c.serverRepo.Update(ctx, server); err != nil {
		logger.Log.Error("Failed to update server status in DB",
			zap.String("serverID", payload.ID),
			zap.String("newStatus", payload.Status),
			zap.Error(err))
		return err
	}

	logger.Log.Info("Updated server status in DB",
		zap.String("serverID", payload.ID),
		zap.String("status", payload.Status))

	return nil
}
