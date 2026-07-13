package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status ServerStatus
		want   bool
	}{
		{"ONLINE is valid", ServerStatusOnline, true},
		{"OFFLINE is valid", ServerStatusOffline, true},
		{"empty string is invalid", ServerStatus(""), false},
		{"UNKNOWN is invalid", ServerStatus("UNKNOWN"), false},
		{"random is invalid", ServerStatus("RANDOM"), false},
		{"lowercase online is invalid", ServerStatus("online"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestServer_TableName(t *testing.T) {
	s := &Server{}
	assert.Equal(t, "management_schema.servers", s.TableName())
}

func TestServer_DefaultValues(t *testing.T) {
	s := &Server{}
	assert.Equal(t, ServerStatus(""), s.CurrentStatus, "zero value of ServerStatus should be empty string")
	assert.Equal(t, 0, s.ConsecutiveFailures, "ConsecutiveFailures should default to 0")
	assert.Empty(t, s.ServerID, "ServerID should be empty before DB insert")
	assert.Empty(t, s.ServerName, "ServerName should be empty")
	assert.Empty(t, s.IPv4, "IPv4 should be empty")
}
