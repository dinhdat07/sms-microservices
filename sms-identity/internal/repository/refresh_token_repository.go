package repository

import (
	"context"
	"github.com/google/uuid"
	"sms-identity/internal/domain"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *domain.RefreshToken) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	RevokeByID(ctx context.Context, id uuid.UUID) error
	RevokeBySessionID(ctx context.Context, sessionID uuid.UUID) error
	RevokeByUserID(ctx context.Context, userID uint) error
	MarkReplacement(ctx context.Context, oldTokenID uuid.UUID, newTokenID uuid.UUID) error
}