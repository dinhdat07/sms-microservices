package repository

import (
	"context"
	"time"
	"github.com/google/uuid"
)

type SessionRevocationStore interface {
	MarkRevoked(ctx context.Context, sessionID uuid.UUID, expiresAt time.Time) error
}