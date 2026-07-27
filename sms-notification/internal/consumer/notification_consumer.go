package consumer

import (
	"context"
	"encoding/json"

	"sms-notification/internal/config"
	"sms-notification/internal/domain"
	"sms-notification/internal/infrastructure/logger"
	"sms-notification/internal/infrastructure/messagebroker"

	"go.uber.org/zap"
)

type NotificationConsumer interface {
	Start(ctx context.Context)
}

type notificationConsumerImpl struct {
	subscriber messagebroker.Subscriber
	cfg        config.Config
	sender     domain.NotificationSender
}

func NewNotificationConsumer(subscriber messagebroker.Subscriber, cfg config.Config, sender domain.NotificationSender) NotificationConsumer {
	return &notificationConsumerImpl{
		subscriber: subscriber,
		cfg:        cfg,
		sender:     sender,
	}
}

func (c *notificationConsumerImpl) Start(ctx context.Context) {
	if c.subscriber == nil {
		logger.Log.Warn("[NotificationConsumer] Subscriber not configured, skipping.")
		return
	}

	logger.Log.Info("[NotificationConsumer] Starting message broker subscriber", zap.String("consumer", c.cfg.ConsumerName))

	go func() {
		if err := c.subscriber.Subscribe(ctx, c.cfg.NotificationStream, c.cfg.NotificationGroup, c.cfg.ConsumerName, c.handleNotificationEvent); err != nil {
			logger.Log.Error("Failed to subscribe to NotificationStream", zap.Error(err))
		}
	}()
}

func (c *notificationConsumerImpl) handleNotificationEvent(ctx context.Context, msg messagebroker.Message) error {
	eventType, ok := msg.Values["event_type"].(string)
	if !ok {
		return nil
	}

	payloadStr, ok := msg.Values["payload"].(string)
	if !ok {
		return nil
	}

	if eventType == domain.EventNotificationRequested {
		var payload domain.NotificationEvent
		if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
			logger.Log.Error("[NotificationConsumer] Failed to unmarshal payload", zap.Error(err))
			return err
		}

		logger.Log.Info("[NotificationConsumer] Sending email", zap.String("to", payload.To), zap.String("subject", payload.Subject))
		
		err := c.sender.SendEmail(ctx, payload.To, payload.Subject, payload.Body)
		if err != nil {
			logger.Log.Error("[NotificationConsumer] Failed to send email", zap.Error(err))
			return err
		}
		
		logger.Log.Info("[NotificationConsumer] Email sent successfully", zap.String("to", payload.To))
	}

	return nil
}
