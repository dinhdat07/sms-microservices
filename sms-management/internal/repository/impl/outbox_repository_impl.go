package impl

import (
	"context"
	"sms-management/internal/domain"
	"sms-management/internal/repository"

	"gorm.io/gorm"
)

type GormOutboxRepository struct {
	db *gorm.DB
}

func NewGormOutboxRepository(db *gorm.DB) repository.OutboxRepository {
	return &GormOutboxRepository{db: db}
}

func (r *GormOutboxRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *GormOutboxRepository) Create(ctx context.Context, event *domain.OutboxEvent) error {
	return r.getDB(ctx).Create(event).Error
}

func (r *GormOutboxRepository) BatchCreate(ctx context.Context, events []*domain.OutboxEvent) error {
	if len(events) == 0 {
		return nil
	}
	return r.getDB(ctx).Create(&events).Error
}

func (r *GormOutboxRepository) GetUnprocessed(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	var events []*domain.OutboxEvent
	err := r.getDB(ctx).Where("is_processed = ?", false).Order("created_at ASC").Limit(limit).Find(&events).Error
	return events, err
}

func (r *GormOutboxRepository) MarkProcessed(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return r.getDB(ctx).Model(&domain.OutboxEvent{}).Where("id IN ?", ids).Update("is_processed", true).Error
}
