package consumer

import (
	"context"
	"encoding/json"
	"testing"

	"sms-notification/internal/config"
	"sms-notification/internal/domain"
	domainMock "sms-notification/internal/domain/mock"
	"sms-notification/internal/infrastructure/messagebroker"
	mbMock "sms-notification/internal/infrastructure/messagebroker/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNotificationConsumer_handleNotificationEvent(t *testing.T) {
	sender := new(domainMock.MockNotificationSender)
	cfg := config.Config{}

	consumer := NewNotificationConsumer(nil, cfg, sender).(*notificationConsumerImpl)
	ctx := context.Background()

	// 1. Success case
	payload := domain.NotificationEvent{
		To:      "test@example.com",
		Subject: "Test Subject",
		Body:    "<p>Test Body</p>",
	}
	payloadBytes, _ := json.Marshal(payload)

	msg := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": domain.EventNotificationRequested,
			"payload":    string(payloadBytes),
		},
	}

	sender.On("SendEmail", ctx, "test@example.com", "Test Subject", "<p>Test Body</p>").Return(nil).Once()

	err := consumer.handleNotificationEvent(ctx, msg)
	assert.NoError(t, err)

	// 2. Ignore invalid event type
	msgInvalidType := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": "UnknownEvent",
			"payload":    string(payloadBytes),
		},
	}
	err = consumer.handleNotificationEvent(ctx, msgInvalidType)
	assert.NoError(t, err) // Should ignore without error

	// 3. Invalid payload
	msgInvalidPayload := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": domain.EventNotificationRequested,
			"payload":    "invalid json",
		},
	}
	err = consumer.handleNotificationEvent(ctx, msgInvalidPayload)
	assert.Error(t, err) // JSON unmarshal error

	// 4. Sender error
	sender.On("SendEmail", ctx, "test@example.com", "Test Subject", "<p>Test Body</p>").Return(assert.AnError).Once()
	err = consumer.handleNotificationEvent(ctx, msg)
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)

	sender.AssertExpectations(t)
	// 5. Missing event_type
	msgMissingType := messagebroker.Message{
		Values: map[string]interface{}{
			"payload": string(payloadBytes),
		},
	}
	err = consumer.handleNotificationEvent(ctx, msgMissingType)
	assert.NoError(t, err)

	// 6. Missing payload
	msgMissingPayload := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": domain.EventNotificationRequested,
		},
	}
	err = consumer.handleNotificationEvent(ctx, msgMissingPayload)
	assert.NoError(t, err)

	sender.AssertExpectations(t)
}

func TestNotificationConsumer_Start(t *testing.T) {
	subscriber := new(mbMock.MockSubscriber)
	cfg := config.Config{
		NotificationStream: "test_stream",
		NotificationGroup:  "test_group",
		ConsumerName:       "test_consumer",
	}

	consumer := NewNotificationConsumer(subscriber, cfg, nil)
	ctx := context.Background()

	subscriber.On("Subscribe", ctx, "test_stream", "test_group", "test_consumer", mock.Anything).Return(nil).Once()

	consumer.Start(ctx)

	// Test nil subscriber
	consumerNil := NewNotificationConsumer(nil, cfg, nil)
	consumerNil.Start(ctx) // Should return early and not panic

	// Test subscribe error
	subscriberErr := new(mbMock.MockSubscriber)
	consumerErr := NewNotificationConsumer(subscriberErr, cfg, nil)
	subscriberErr.On("Subscribe", ctx, "test_stream", "test_group", "test_consumer", mock.Anything).Return(assert.AnError).Once()
	consumerErr.Start(ctx)
}
