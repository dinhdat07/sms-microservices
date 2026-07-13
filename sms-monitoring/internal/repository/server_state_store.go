package repository

import (
	"context"
	"sms-monitoring/internal/domain"
	"errors"
)

var ErrServerStateNotFound = errors.New("server state not found")

// ServerStateStore abstracts the fast cache (Redis) operations for monitoring state.
type ServerStateStore interface {
	GetServerState(ctx context.Context, serverID string) (*domain.ServerState, error)
	SetServerState(ctx context.Context, serverID string, status string, retryCount int) error
}


