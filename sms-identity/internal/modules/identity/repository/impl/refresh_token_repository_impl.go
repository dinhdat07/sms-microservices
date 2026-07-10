package impl

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"sms-identity/internal/modules/identity/domain"
	"sms-identity/internal/modules/identity/repository"
)

type refreshTokenRepoImpl struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) repository.RefreshTokenRepository {
	return &refreshTokenRepoImpl{db: db}
}

func (r *refreshTokenRepoImpl) Create(ctx context.Context, token *domain.RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *refreshTokenRepoImpl) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	var token domain.RefreshToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *refreshTokenRepoImpl) RevokeByID(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&domain.RefreshToken{}).
		Where("id = ?", id).
		Update("revoked_at", time.Now()).Error
}

func (r *refreshTokenRepoImpl) RevokeBySessionID(ctx context.Context, sessionID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&domain.RefreshToken{}).
		Where("session_id = ? AND revoked_at IS NULL", sessionID).
		Update("revoked_at", time.Now()).Error
}

func (r *refreshTokenRepoImpl) RevokeByUserID(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).
		Model(&domain.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now()).Error
}

func (r *refreshTokenRepoImpl) MarkReplacement(ctx context.Context, oldTokenID uuid.UUID, newTokenID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&domain.RefreshToken{}).
		Where("id = ?", oldTokenID).
		Updates(map[string]interface{}{
			"revoked_at":  time.Now(),
			"replaced_by": newTokenID,
		}).Error
}