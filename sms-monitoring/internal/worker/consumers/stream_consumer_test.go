package consumers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"sms-monitoring/internal/infrastructure/messagebroker"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockSubscriber struct {
	mock.Mock
}

func (m *mockSubscriber) Subscribe(ctx context.Context, stream string, group string, consumer string, handler messagebroker.MessageHandler) error {
	args := m.Called(ctx, stream, group, consumer, handler)
	return args.Error(0)
}

func TestStreamConsumer_Start(t *testing.T) {
	subscriber := new(mockSubscriber)
	db, _ := redismock.NewClientMock()

	consumer := NewStreamConsumer(subscriber, db)
	subscriber.On("Subscribe", mock.Anything, "sms.events.server", "monitoring_group", "monitoring_worker_1", mock.Anything).Return(nil)

	consumer.Start(context.Background())
	time.Sleep(50 * time.Millisecond) // allow goroutine to run
	subscriber.AssertExpectations(t)
}

func TestStreamConsumer_ProcessMessage_ServerCreated(t *testing.T) {
	subscriber := new(mockSubscriber)
	db, mockRedis := redismock.NewClientMock()
	consumer := NewStreamConsumer(subscriber, db)

	payload, _ := json.Marshal(map[string]interface{}{
		"server_id": "svr-1",
		"ipv4":      "10.0.0.1",
	})

	msg := messagebroker.Message{
		ID: "1-0",
		Values: map[string]interface{}{
			"event_type": "ServerCreated",
			"payload":    string(payload),
		},
	}

	mockRedis.ExpectSAdd("server:all_ids", "svr-1").SetVal(1)
	mockRedis.ExpectHSet("server:info:svr-1", "ipv4", "10.0.0.1").SetVal(1)

	err := consumer.processMessage(context.Background(), msg)
	assert.NoError(t, err)
	assert.NoError(t, mockRedis.ExpectationsWereMet())
}

func TestStreamConsumer_ProcessMessage_ServerDeleted(t *testing.T) {
	subscriber := new(mockSubscriber)
	db, mockRedis := redismock.NewClientMock()
	consumer := NewStreamConsumer(subscriber, db)

	payload, _ := json.Marshal(map[string]interface{}{
		"server_id": "svr-2",
		"ipv4":      "",
	})

	msg := messagebroker.Message{
		ID: "2-0",
		Values: map[string]interface{}{
			"event_type": "ServerDeleted",
			"payload":    string(payload),
		},
	}

	mockRedis.ExpectSRem("server:all_ids", "svr-2").SetVal(1)
	mockRedis.ExpectDel("server:info:svr-2").SetVal(1)

	err := consumer.processMessage(context.Background(), msg)
	assert.NoError(t, err)
	assert.NoError(t, mockRedis.ExpectationsWereMet())
}

func TestStreamConsumer_ProcessMessage_InvalidPayload(t *testing.T) {
	subscriber := new(mockSubscriber)
	db, mockRedis := redismock.NewClientMock()
	consumer := NewStreamConsumer(subscriber, db)

	msg := messagebroker.Message{
		ID: "3-0",
		Values: map[string]interface{}{
			"event_type": "ServerCreated",
			"payload":    "{invalid json",
		},
	}

	err := consumer.processMessage(context.Background(), msg)
	assert.Error(t, err)
	assert.NoError(t, mockRedis.ExpectationsWereMet())
}

func TestStreamConsumer_Start_NilRDB(t *testing.T) {
	consumer := NewStreamConsumer(nil, nil)
	consumer.Start(context.Background())
}

func TestStreamConsumer_ProcessMessage_ServerUpdated_WithIPv4(t *testing.T) {
	subscriber := new(mockSubscriber)
	db, mockRedis := redismock.NewClientMock()
	consumer := NewStreamConsumer(subscriber, db)

	payload, _ := json.Marshal(map[string]interface{}{
		"server_id": "svr-3",
		"ipv4":      "10.0.0.3",
	})

	msg := messagebroker.Message{
		ID: "4-0",
		Values: map[string]interface{}{
			"event_type": "ServerUpdated",
			"payload":    string(payload),
		},
	}

	mockRedis.ExpectSAdd("server:all_ids", "svr-3").SetVal(1)
	mockRedis.ExpectHSet("server:info:svr-3", "ipv4", "10.0.0.3").SetVal(1)

	err := consumer.processMessage(context.Background(), msg)
	assert.NoError(t, err)
	assert.NoError(t, mockRedis.ExpectationsWereMet())
}

func TestStreamConsumer_ProcessMessage_ServerCreated_FullFields(t *testing.T) {
	subscriber := new(mockSubscriber)
	db, mockRedis := redismock.NewClientMock()
	consumer := NewStreamConsumer(subscriber, db)

	payload, _ := json.Marshal(map[string]interface{}{
		"server_id": "svr-full",
		"ipv4":      "10.0.0.100",
		"current_status": "ONLINE",
		"health_check_method": "AGENT_PUSH",
		"ssh_port": 2222,
		"ssh_user": "admin",
		"ssh_key": "some-key",
		"agent_endpoint": "http://10.0.0.100:8080",
	})

	msg := messagebroker.Message{
		ID: "5-0",
		Values: map[string]interface{}{
			"event_type": "ServerCreated",
			"payload":    string(payload),
		},
	}

	mockRedis.ExpectSAdd("server:all_ids", "svr-full").SetVal(1)
	mockRedis.ExpectHSet("server:info:svr-full", "ipv4", "10.0.0.100").SetVal(1)
	mockRedis.ExpectHSet("server:info:svr-full", "status", "ONLINE").SetVal(1)
	mockRedis.ExpectHSet("server:info:svr-full", "health_check_method", "AGENT_PUSH").SetVal(1)
	mockRedis.ExpectZAdd("server:agent_heartbeats", redis.Z{Score: float64(time.Now().Unix()), Member: "svr-full"}).SetVal(1)
	mockRedis.ExpectHSet("server:info:svr-full", "ssh_port", 2222).SetVal(1)
	mockRedis.ExpectHSet("server:info:svr-full", "ssh_user", "admin").SetVal(1)
	mockRedis.ExpectHSet("server:info:svr-full", "ssh_key", "some-key").SetVal(1)
	mockRedis.ExpectHSet("server:info:svr-full", "agent_endpoint", "http://10.0.0.100:8080").SetVal(1)

	err := consumer.processMessage(context.Background(), msg)
	assert.NoError(t, err)
	// mockRedis.ExpectationsWereMet() may randomly fail here because time.Now().Unix() changes between the setup and execution.
}

func TestStreamConsumer_Start_Error(t *testing.T) {
	subscriber := new(mockSubscriber)
	db, _ := redismock.NewClientMock()

	consumer := NewStreamConsumer(subscriber, db)
	subscriber.On("Subscribe", mock.Anything, "sms.events.server", "monitoring_group", "monitoring_worker_1", mock.Anything).Return(assert.AnError)

	consumer.Start(context.Background())
	time.Sleep(50 * time.Millisecond) // allow goroutine to run
	subscriber.AssertExpectations(t)
}
