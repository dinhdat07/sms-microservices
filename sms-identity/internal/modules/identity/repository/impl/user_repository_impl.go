package impl

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"sms-identity/internal/modules/identity/domain"
	"sms-identity/internal/modules/identity/repository"
)

type userRepositoryImpl struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) repository.UserRepository {
	return &userRepositoryImpl{db: db}
}

func (r *userRepositoryImpl) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil, nil when not found to easily handle it in service
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepositoryImpl) FindByID(ctx context.Context, id uint) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}