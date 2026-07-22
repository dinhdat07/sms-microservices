package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"sms-management/internal/config"
	"sms-management/internal/domain"
	brokerMock "sms-management/internal/infrastructure/messagebroker/mock"
	repoMock "sms-management/internal/repository/mock"

	"github.com/stretchr/testify/mock"

)

func TestOutboxRelay_processOutboxEvents(t *testing.T) {
	mockRepo := new(repoMock.MockOutboxRepository)
	mockPub := new(brokerMock.MockPublisher)
	cfg := config.OutboxConfig{BatchSize: 100, StreamName: "outbox_events", IntervalMs: 100}
	relay := NewOutboxRelay(mockRepo, mockPub, cfg)

	ctx := context.Background()

	// 1. No events
	mockRepo.On("GetUnprocessed", ctx, 100).Return([]*domain.OutboxEvent{}, nil).Once()
	relay.processOutboxEvents()
	
	// 2. Fetch error
	mockRepo.On("GetUnprocessed", ctx, 100).Return(nil, errors.New("db error")).Once()
	relay.processOutboxEvents()

	// 3. Process events successfully
	events := []*domain.OutboxEvent{
		{ID: "evt-1", AggregateType: "Server", AggregateID: "srv-1", EventType: "ServerCreated", Payload: []byte(`{}`)},
		{ID: "evt-2", AggregateType: "Server", AggregateID: "srv-2", EventType: "ServerUpdated", Payload: []byte(`{}`)},
	}
	
	mockRepo.On("GetUnprocessed", ctx, 100).Return(events, nil).Once()
	mockPub.On("PublishOutboxBatch", ctx, "outbox_events", events).Return(nil).Once()
	mockRepo.On("MarkProcessed", ctx, []string{"evt-1", "evt-2"}).Return(nil).Once()

	relay.processOutboxEvents()

	// 4. Publish error
	mockRepo.On("GetUnprocessed", ctx, 100).Return(events, nil).Once()
	mockPub.On("PublishOutboxBatch", ctx, "outbox_events", events).Return(errors.New("redis error")).Once()
	
	relay.processOutboxEvents()
}

func TestOutboxRelay_StartStop(t *testing.T) {
	mockRepo := new(repoMock.MockOutboxRepository)
	mockPub := new(brokerMock.MockPublisher)
	cfg := config.OutboxConfig{BatchSize: 10, StreamName: "outbox", IntervalMs: 10}
	relay := NewOutboxRelay(mockRepo, mockPub, cfg)

	mockRepo.On("GetUnprocessed", mock.Anything, 10).Return([]*domain.OutboxEvent{}, nil)

	// Start the relay
	relay.Start()

	// Let it run briefly
	time.Sleep(50 * time.Millisecond)

	// Stop it
	relay.Stop()
}
