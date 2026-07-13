package repository

import (
	"context"
	"sms-management/internal/domain"
)

type OutboxRepository interface {
	Create(ctx context.Context, event *domain.OutboxEvent) error
	BatchCreate(ctx context.Context, events []*domain.OutboxEvent) error
	GetUnprocessed(ctx context.Context, limit int) ([]*domain.OutboxEvent, error)
	MarkProcessed(ctx context.Context, ids []string) error
}
