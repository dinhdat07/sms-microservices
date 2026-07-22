package consumer

import (
	"context"
	"encoding/json"
	"testing"

	"sms-reporting/internal/config"
	"sms-reporting/internal/domain"
	"sms-reporting/internal/infrastructure/messagebroker"
	brokerMock "sms-reporting/internal/infrastructure/messagebroker/mock"
	repoMock "sms-reporting/internal/repository/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestEventConsumer_Start(t *testing.T) {
	subscriber := new(brokerMock.MockSubscriber)
	repo := new(repoMock.MockReportingRepository)
	cfg := config.ConsumerConfig{
		Name:               "test_consumer",
		ServerStream:       "sms.events.server",
		ServerGroup:        "reporting_group",
		ServerStatusStream: "sms.events.server_status",
		ServerStatusGroup:  "reporting_status_group",
	}

	consumer := NewEventConsumer(subscriber, repo, cfg)

	subscriber.On("Subscribe", mock.Anything, "sms.events.server", "reporting_group", "test_consumer", mock.Anything).Return(nil)
	subscriber.On("Subscribe", mock.Anything, "sms.events.server_status", "reporting_status_group", "test_consumer", mock.Anything).Return(nil)

	consumer.Start(context.Background())

	// Cannot easily assert goroutine execution in Start, but we can verify no panics and mocks eventually called
}

func TestEventConsumer_HandleServerEvent_Created(t *testing.T) {
	subscriber := new(brokerMock.MockSubscriber)
	repo := new(repoMock.MockReportingRepository)
	consumer := NewEventConsumer(subscriber, repo, config.ConsumerConfig{}).(*eventConsumerImpl)

	payload, _ := json.Marshal(map[string]interface{}{
		"server_id":   "svr-1",
		"server_name": "Test Server",
		"ipv4":        "10.0.0.1",
	})

	msg := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": "ServerCreated",
			"payload":    string(payload),
		},
	}

	repo.On("UpsertReportingServer", context.Background(), &domain.ReportingServer{
		ServerID: "svr-1",
		Name:     "Test Server",
		IPv4:     "10.0.0.1",
		Status:   "ONLINE",
	}).Return(nil)

	err := consumer.handleServerEvent(context.Background(), msg)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestEventConsumer_HandleServerEvent_Deleted(t *testing.T) {
	subscriber := new(brokerMock.MockSubscriber)
	repo := new(repoMock.MockReportingRepository)
	consumer := NewEventConsumer(subscriber, repo, config.ConsumerConfig{}).(*eventConsumerImpl)

	payload, _ := json.Marshal(map[string]interface{}{
		"server_id": "svr-2",
	})

	msg := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": "ServerDeleted",
			"payload":    string(payload),
		},
	}

	repo.On("DeleteReportingServer", context.Background(), "svr-2").Return(nil)

	err := consumer.handleServerEvent(context.Background(), msg)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestEventConsumer_HandleStatusEvent(t *testing.T) {
	subscriber := new(brokerMock.MockSubscriber)
	repo := new(repoMock.MockReportingRepository)
	consumer := NewEventConsumer(subscriber, repo, config.ConsumerConfig{}).(*eventConsumerImpl)

	payload, _ := json.Marshal(map[string]interface{}{
		"server_id": "svr-3",
		"status":    "OFFLINE",
	})

	msg := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": "ServerStatusChanged",
			"payload":    string(payload),
		},
	}

	repo.On("UpdateReportingServerStatus", context.Background(), "svr-3", "OFFLINE").Return(nil)

	err := consumer.handleStatusEvent(context.Background(), msg)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestEventConsumer_HandleServerEvent_InvalidEventType(t *testing.T) {
	subscriber := new(brokerMock.MockSubscriber)
	repo := new(repoMock.MockReportingRepository)
	consumer := NewEventConsumer(subscriber, repo, config.ConsumerConfig{}).(*eventConsumerImpl)

	msg := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": "InvalidEvent",
			"payload":    "{}",
		},
	}

	err := consumer.handleServerEvent(context.Background(), msg)
	assert.NoError(t, err) // Invalid event is ignored, no error returned
}

func TestEventConsumer_HandleServerEvent_MissingPayload(t *testing.T) {
	subscriber := new(brokerMock.MockSubscriber)
	repo := new(repoMock.MockReportingRepository)
	consumer := NewEventConsumer(subscriber, repo, config.ConsumerConfig{}).(*eventConsumerImpl)

	msg := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": "ServerCreated",
		},
	}

	err := consumer.handleServerEvent(context.Background(), msg)
	assert.NoError(t, err) // Missing payload
}

func TestEventConsumer_HandleServerEvent_InvalidJSON(t *testing.T) {
	subscriber := new(brokerMock.MockSubscriber)
	repo := new(repoMock.MockReportingRepository)
	consumer := NewEventConsumer(subscriber, repo, config.ConsumerConfig{}).(*eventConsumerImpl)

	msg := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": "ServerCreated",
			"payload":    "{invalid json",
		},
	}

	err := consumer.handleServerEvent(context.Background(), msg)
	assert.Error(t, err)
}

func TestEventConsumer_HandleStatusEvent_MissingPayload(t *testing.T) {
	subscriber := new(brokerMock.MockSubscriber)
	repo := new(repoMock.MockReportingRepository)
	consumer := NewEventConsumer(subscriber, repo, config.ConsumerConfig{}).(*eventConsumerImpl)

	msg := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": "ServerStatusChanged",
		},
	}

	err := consumer.handleStatusEvent(context.Background(), msg)
	assert.NoError(t, err)
}

func TestEventConsumer_HandleServerEvent_Created_DBError(t *testing.T) {
	subscriber := new(brokerMock.MockSubscriber)
	repo := new(repoMock.MockReportingRepository)
	consumer := NewEventConsumer(subscriber, repo, config.ConsumerConfig{}).(*eventConsumerImpl)

	payload, _ := json.Marshal(map[string]interface{}{
		"server_id":   "svr-1",
		"server_name": "Test Server",
		"ipv4":        "10.0.0.1",
	})

	msg := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": "ServerCreated",
			"payload":    string(payload),
		},
	}

	repo.On("UpsertReportingServer", context.Background(), mock.Anything).Return(assert.AnError)

	err := consumer.handleServerEvent(context.Background(), msg)
	assert.Error(t, err)
}

func TestEventConsumer_HandleServerEvent_Deleted_DBError(t *testing.T) {
	subscriber := new(brokerMock.MockSubscriber)
	repo := new(repoMock.MockReportingRepository)
	consumer := NewEventConsumer(subscriber, repo, config.ConsumerConfig{}).(*eventConsumerImpl)

	payload, _ := json.Marshal(map[string]interface{}{
		"server_id": "svr-2",
	})

	msg := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": "ServerDeleted",
			"payload":    string(payload),
		},
	}

	repo.On("DeleteReportingServer", context.Background(), "svr-2").Return(assert.AnError)

	err := consumer.handleServerEvent(context.Background(), msg)
	assert.Error(t, err)
}

func TestEventConsumer_HandleStatusEvent_DBError(t *testing.T) {
	subscriber := new(brokerMock.MockSubscriber)
	repo := new(repoMock.MockReportingRepository)
	consumer := NewEventConsumer(subscriber, repo, config.ConsumerConfig{}).(*eventConsumerImpl)

	payload, _ := json.Marshal(map[string]interface{}{
		"server_id": "svr-3",
		"status":    "OFFLINE",
	})

	msg := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": "ServerStatusChanged",
			"payload":    string(payload),
		},
	}

	repo.On("UpdateReportingServerStatus", context.Background(), "svr-3", "OFFLINE").Return(assert.AnError)

	err := consumer.handleStatusEvent(context.Background(), msg)
	assert.Error(t, err)
}

func TestEventConsumer_HandleStatusEvent_InvalidJSON(t *testing.T) {
	subscriber := new(brokerMock.MockSubscriber)
	repo := new(repoMock.MockReportingRepository)
	consumer := NewEventConsumer(subscriber, repo, config.ConsumerConfig{}).(*eventConsumerImpl)

	msg := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": "ServerStatusChanged",
			"payload":    "{invalid",
		},
	}

	err := consumer.handleStatusEvent(context.Background(), msg)
	assert.Error(t, err)
}
