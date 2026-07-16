package consumer

import (
	"context"
	"encoding/json"

	"sms-reporting/internal/config"
	"sms-reporting/internal/domain"
	"sms-reporting/internal/infrastructure/logger"
	"sms-reporting/internal/infrastructure/messagebroker"
	"sms-reporting/internal/repository"

	"go.uber.org/zap"
)

type EventConsumer interface {
	Start(ctx context.Context)
}

type eventConsumerImpl struct {
	subscriber messagebroker.Subscriber
	repo       repository.ReportingRepository
	cfg        config.ConsumerConfig
}

func NewEventConsumer(subscriber messagebroker.Subscriber, repo repository.ReportingRepository, cfg config.ConsumerConfig) EventConsumer {
	return &eventConsumerImpl{
		subscriber: subscriber,
		repo:       repo,
		cfg:        cfg,
	}
}

func (c *eventConsumerImpl) Start(ctx context.Context) {
	if c.subscriber == nil {
		logger.Log.Warn("[EventConsumer] Subscriber not configured, skipping consumer.")
		return
	}

	logger.Log.Info("[EventConsumer] Starting message broker subscriber", zap.String("consumer", c.cfg.Name))

	go c.subscriber.Subscribe(ctx, c.cfg.ServerStream, c.cfg.ServerGroup, c.cfg.Name, c.handleServerEvent)
	go c.subscriber.Subscribe(ctx, c.cfg.ServerStatusStream, c.cfg.ServerStatusGroup, c.cfg.Name, c.handleStatusEvent)
}

func (c *eventConsumerImpl) handleServerEvent(ctx context.Context, msg messagebroker.Message) error {
	eventType, ok := msg.Values["event_type"].(string)
	if !ok {
		return nil
	}

	payloadStr, ok := msg.Values["payload"].(string)
	if !ok {
		return nil
	}

	switch eventType {
	case "ServerCreated", "ServerUpdated":
		var payload struct {
			ID   string `json:"server_id"`
			Name string `json:"server_name"`
			IP   string `json:"ipv4"`
		}
		if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
			logger.Log.Error("[EventConsumer] Failed to unmarshal Server event payload", zap.Error(err))
			return err
		}

		server := &domain.ReportingServer{
			ServerID: payload.ID,
			Name:     payload.Name,
			IPv4:     payload.IP,
			Status:   "ONLINE", // Default status for new servers
		}
		err := c.repo.UpsertReportingServer(ctx, server)
		if err != nil {
			logger.Log.Error("[EventConsumer] Failed to upsert reporting server", zap.Error(err))
			return err
		}
		logger.Log.Info("[EventConsumer] Processed Server event", zap.String("eventType", eventType), zap.String("serverID", payload.ID))

	case "ServerDeleted":
		var payload struct {
			ID string `json:"server_id"`
		}
		if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
			logger.Log.Error("[EventConsumer] Failed to unmarshal ServerDeleted payload", zap.Error(err))
			return err
		}

		err := c.repo.DeleteReportingServer(ctx, payload.ID)
		if err != nil {
			logger.Log.Error("[EventConsumer] Failed to delete reporting server", zap.Error(err))
			return err
		}
		logger.Log.Info("[EventConsumer] Processed ServerDeleted event", zap.String("serverID", payload.ID))
	}

	return nil
}

func (c *eventConsumerImpl) handleStatusEvent(ctx context.Context, msg messagebroker.Message) error {
	eventType, ok := msg.Values["event_type"].(string)
	if !ok {
		return nil
	}

	payloadStr, ok := msg.Values["payload"].(string)
	if !ok {
		return nil
	}

	if eventType == "ServerStatusChanged" {
		var payload struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		}
		if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
			logger.Log.Error("[EventConsumer] Failed to unmarshal ServerStatusChanged payload", zap.Error(err))
			return err
		}

		err := c.repo.UpdateReportingServerStatus(ctx, payload.ID, payload.Status)
		if err != nil {
			logger.Log.Error("[EventConsumer] Failed to update reporting server status", zap.Error(err))
			return err
		}
		logger.Log.Info("[EventConsumer] Processed ServerStatusChanged event", zap.String("serverID", payload.ID), zap.String("status", payload.Status))
	}

	return nil
}
