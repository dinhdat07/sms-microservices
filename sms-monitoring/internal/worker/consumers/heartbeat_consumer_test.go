package consumers

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"sms-monitoring/internal/infrastructure/messagebroker"
	infraRedis "sms-monitoring/internal/infrastructure/redis"
	"sms-monitoring/internal/service/mock"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)



func TestHeartbeatConsumer_Start(t *testing.T) {
	mockSub := new(mockSubscriber)
	mockMonService := mocks.NewMonitoringService(t)

	t.Run("redis nil", func(t *testing.T) {
		consumer := NewHeartbeatConsumer(mockSub, mockMonService, nil)
		consumer.Start(context.Background())
		mockSub.AssertNotCalled(t, "Subscribe")
	})

	t.Run("start success", func(t *testing.T) {
		mr, _ := miniredis.Run()
		defer mr.Close()
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

		mockSub.On("Subscribe", mock.Anything, "sms.events.heartbeat", "monitoring_group", "monitoring_worker_heartbeat", mock.Anything).Return(nil).Once()

		consumer := NewHeartbeatConsumer(mockSub, mockMonService, rdb)
		consumer.Start(context.Background())
		
		time.Sleep(50 * time.Millisecond) // wait for goroutine

		mockSub.AssertExpectations(t)
	})
	
	t.Run("start error log", func(t *testing.T) {
		mr, _ := miniredis.Run()
		defer mr.Close()
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

		mockSub.On("Subscribe", mock.Anything, "sms.events.heartbeat", "monitoring_group", "monitoring_worker_heartbeat", mock.Anything).Return(errors.New("sub error")).Once()

		consumer := NewHeartbeatConsumer(mockSub, mockMonService, rdb)
		consumer.Start(context.Background())
		
		time.Sleep(50 * time.Millisecond) // wait for goroutine

		mockSub.AssertExpectations(t)
	})
}

func TestHeartbeatConsumer_processMessage(t *testing.T) {
	mockSub := new(mockSubscriber)
	
	t.Run("missing server id", func(t *testing.T) {
		mockMonService := mocks.NewMonitoringService(t)
		mr, _ := miniredis.Run()
		defer mr.Close()
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

		consumer := NewHeartbeatConsumer(mockSub, mockMonService, rdb)
		err := consumer.processMessage(context.Background(), messagebroker.Message{
			Values: map[string]interface{}{}, // missing
		})

		assert.NoError(t, err)
		mockMonService.AssertNotCalled(t, "Evaluate")
	})

	t.Run("redis hget success", func(t *testing.T) {
		mockMonService := mocks.NewMonitoringService(t)
		mr, _ := miniredis.Run()
		defer mr.Close()
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		
		mr.HSet(fmt.Sprintf(infraRedis.ServerInfoKeyFmt, "srv-1"), infraRedis.ServerInfoFieldIPv4, "1.2.3.4")

		consumer := NewHeartbeatConsumer(mockSub, mockMonService, rdb)

		mockMonService.On("Evaluate", mock.Anything, "srv-1", "1.2.3.4", true).Return(nil).Once()

		err := consumer.processMessage(context.Background(), messagebroker.Message{
			Values: map[string]interface{}{
				"server_id": "srv-1",
			},
		})

		assert.NoError(t, err)
		mockMonService.AssertExpectations(t)
	})

	t.Run("redis hget fails fallback", func(t *testing.T) {
		mockMonService := mocks.NewMonitoringService(t)
		mr, _ := miniredis.Run()
		defer mr.Close()
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		// No HSet, it will fallback to empty string

		consumer := NewHeartbeatConsumer(mockSub, mockMonService, rdb)

		mockMonService.On("Evaluate", mock.Anything, "srv-1", "", true).Return(nil).Once()

		err := consumer.processMessage(context.Background(), messagebroker.Message{
			Values: map[string]interface{}{
				"server_id": "srv-1",
			},
		})

		assert.NoError(t, err)
		mockMonService.AssertExpectations(t)
	})

	t.Run("evaluate fails", func(t *testing.T) {
		mockMonService := mocks.NewMonitoringService(t)
		mr, _ := miniredis.Run()
		defer mr.Close()
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

		consumer := NewHeartbeatConsumer(mockSub, mockMonService, rdb)

		mockErr := errors.New("eval error")
		mockMonService.On("Evaluate", mock.Anything, "srv-1", "", true).Return(mockErr).Once()

		err := consumer.processMessage(context.Background(), messagebroker.Message{
			Values: map[string]interface{}{
				"server_id": "srv-1",
			},
		})

		assert.ErrorIs(t, err, mockErr)
		mockMonService.AssertExpectations(t)
	})
}
