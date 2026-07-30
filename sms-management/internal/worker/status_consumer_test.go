package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"sms-management/internal/domain"
	"sms-management/internal/infrastructure/messagebroker"
	brokerMock "sms-management/internal/infrastructure/messagebroker/mock"
	repoMock "sms-management/internal/repository/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStatusConsumer_Start(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockSub := new(brokerMock.MockSubscriber)
		mockRepo := new(repoMock.MockServerRepository)

		consumer := NewStatusConsumer(mockSub, mockRepo)

		mockSub.On("Subscribe", mock.Anything, "sms.events.server_status", "management_group", "management_worker_1", mock.Anything).Return(nil)

		consumer.Start(context.Background())
	})

	t.Run("error", func(t *testing.T) {
		mockSub := new(brokerMock.MockSubscriber)
		mockRepo := new(repoMock.MockServerRepository)

		consumer := NewStatusConsumer(mockSub, mockRepo)

		mockSub.On("Subscribe", mock.Anything, "sms.events.server_status", "management_group", "management_worker_1", mock.Anything).Return(errors.New("subscribe error"))

		consumer.Start(context.Background())
	})
}

func TestStatusConsumer_processMessage(t *testing.T) {
	mockRepo := new(repoMock.MockServerRepository)
	consumer := NewStatusConsumer(nil, mockRepo)

	ctx := context.Background()

	// 1. Not a ServerStatusChanged event
	msg1 := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": "OtherEvent",
		},
	}
	err := consumer.processMessage(ctx, msg1)
	assert.NoError(t, err)

	// 2. Missing payload
	msg2 := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": "ServerStatusChanged",
		},
	}
	err = consumer.processMessage(ctx, msg2)
	assert.NoError(t, err)

	// 3. Valid event, update server
	payload, _ := json.Marshal(map[string]interface{}{
		"server_id":     "svr-1",
		"status": "OFFLINE",
	})
	msg3 := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": "ServerStatusChanged",
			"payload":    string(payload),
		},
	}
	
	existingServer := &domain.Server{
		ServerID:      "svr-1",
		CurrentStatus: domain.ServerStatusOnline,
	}
	
	mockRepo.On("GetByID", ctx, "svr-1").Return(existingServer, nil).Once()
	mockRepo.On("Update", ctx, mock.AnythingOfType("*domain.Server")).Return(nil).Once()

	err = consumer.processMessage(ctx, msg3)
	assert.NoError(t, err)
	assert.Equal(t, domain.ServerStatusOffline, existingServer.CurrentStatus)

	// 4. Server not found (should not error, just debug log)
	mockRepo.On("GetByID", ctx, "svr-2").Return(nil, errors.New("not found")).Once()
	payload2, _ := json.Marshal(map[string]interface{}{
		"server_id":     "svr-2",
		"status": "OFFLINE",
	})
	msg4 := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": "ServerStatusChanged",
			"payload":    string(payload2),
		},
	}
	err = consumer.processMessage(ctx, msg4)
	assert.NoError(t, err)

	// 5. Update fails
	mockRepo.On("GetByID", ctx, "svr-3").Return(&domain.Server{ServerID: "svr-3", CurrentStatus: domain.ServerStatusOnline}, nil).Once()
	mockRepo.On("Update", ctx, mock.Anything).Return(errors.New("db error")).Once()
	payload3, _ := json.Marshal(map[string]interface{}{
		"server_id":     "svr-3",
		"status": "OFFLINE",
	})
	msg5 := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": "ServerStatusChanged",
			"payload":    string(payload3),
		},
	}
	err = consumer.processMessage(ctx, msg5)
	assert.Error(t, err)

	// 6. event_type cast fails
	msg6 := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": 123,
		},
	}
	err = consumer.processMessage(ctx, msg6)
	assert.NoError(t, err)

	// 7. unmarshal fails
	msg7 := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": "ServerStatusChanged",
			"payload":    `{invalid json`,
		},
	}
	err = consumer.processMessage(ctx, msg7)
	assert.Error(t, err)

	// 8. missing id or status
	payload8, _ := json.Marshal(map[string]interface{}{
		"server_id": "",
		"status":    "",
	})
	msg8 := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": "ServerStatusChanged",
			"payload":    string(payload8),
		},
	}
	err = consumer.processMessage(ctx, msg8)
	assert.NoError(t, err)

	// 9. no change needed
	mockRepo.On("GetByID", ctx, "svr-4").Return(&domain.Server{ServerID: "svr-4", CurrentStatus: domain.ServerStatusOnline}, nil).Once()
	payload9, _ := json.Marshal(map[string]interface{}{
		"server_id": "svr-4",
		"status":    "ONLINE",
	})
	msg9 := messagebroker.Message{
		Values: map[string]interface{}{
			"event_type": "ServerStatusChanged",
			"payload":    string(payload9),
		},
	}
	err = consumer.processMessage(ctx, msg9)
	assert.NoError(t, err)
}
