package impl

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"sms-identity/internal/modules/identity/domain"
	"sms-identity/internal/modules/identity/repository"
)

type authSessionRepoImpl struct {
	db *gorm.DB
}

func NewAuthSessionRepository(db *gorm.DB) repository.AuthSessionRepository {
	return &authSessionRepoImpl{db: db}
}

func (r *authSessionRepoImpl) Create(ctx context.Context, session *domain.AuthSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *authSessionRepoImpl) FindActiveByID(ctx context.Context, id uuid.UUID) (*domain.AuthSession, error) {
	var session domain.AuthSession
	err := r.db.WithContext(ctx).
		Where("id = ? AND revoked_at IS NULL AND expires_at > ?", id, time.Now()).
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *authSessionRepoImpl) ListActiveByUserID(ctx context.Context, userID uint) ([]*domain.AuthSession, error) {
	var sessions []*domain.AuthSession
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now()).
		Find(&sessions).Error
	return sessions, err
}

func (r *authSessionRepoImpl) RevokeByID(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&domain.AuthSession{}).
		Where("id = ?", id).
		Update("revoked_at", time.Now()).Error
}

func (r *authSessionRepoImpl) RevokeAllByUserID(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).
		Model(&domain.AuthSession{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now()).Error
}