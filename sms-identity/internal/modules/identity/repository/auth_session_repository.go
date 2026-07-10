package repository

import (
	"context"
	"github.com/google/uuid"
	"sms-identity/internal/modules/identity/domain"
)

type AuthSessionRepository interface {
	Create(ctx context.Context, session *domain.AuthSession) error
	FindActiveByID(ctx context.Context, id uuid.UUID) (*domain.AuthSession, error)
	ListActiveByUserID(ctx context.Context, userID uint) ([]*domain.AuthSession, error)
	RevokeByID(ctx context.Context, id uuid.UUID) error
	RevokeAllByUserID(ctx context.Context, userID uint) error
}