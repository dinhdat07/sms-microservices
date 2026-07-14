package messagebroker

import (
	"context"

	"sms-management/internal/domain"
)

type Publisher interface {
	PublishOutboxBatch(ctx context.Context, stream string, events []*domain.OutboxEvent) error
}
