package domain_test

import (
	"testing"
	"sms-management/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestNewOutboxEvent(t *testing.T) {
	evt := domain.NewOutboxEvent("Test", "1", "TestEvent", []byte(`{}`))
	assert.NotEmpty(t, evt.ID)
	assert.Equal(t, "Test", evt.AggregateType)
	assert.Equal(t, "1", evt.AggregateID)
	assert.Equal(t, domain.EventType("TestEvent"), evt.EventType)
	assert.False(t, evt.IsProcessed)
	assert.NotZero(t, evt.CreatedAt)
}

func TestServerValidation(t *testing.T) {
	srv := &domain.Server{
		ServerName: "test-server",
		IPv4:       "10.0.0.1",
	}
	assert.Equal(t, "test-server", srv.ServerName)
	assert.Equal(t, "10.0.0.1", srv.IPv4)
}
