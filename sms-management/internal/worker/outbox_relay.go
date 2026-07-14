package worker

import (
	"context"
	"time"

	"sms-management/internal/config"
	"sms-management/internal/domain"
	"sms-management/internal/infrastructure/logger"
	"sms-management/internal/infrastructure/messagebroker"
	"sms-management/internal/repository"
)

type OutboxRelay struct {
	repo       repository.OutboxRepository
	publisher  messagebroker.Publisher
	streamName string
	batchSize  int
	interval   time.Duration
	stopCh     chan struct{}
}

func NewOutboxRelay(repo repository.OutboxRepository, publisher messagebroker.Publisher, cfg config.OutboxConfig) *OutboxRelay {
	return &OutboxRelay{
		repo:       repo,
		publisher:  publisher,
		streamName: cfg.StreamName,
		batchSize:  cfg.BatchSize,
		interval:   time.Duration(cfg.IntervalMs) * time.Millisecond,
		stopCh:     make(chan struct{}),
	}
}

func (r *OutboxRelay) Start() {
	go func() {
		logger.Log.Info("Starting Outbox Relay Worker...")
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()

		for {
			select {
			case <-r.stopCh:
				logger.Log.Info("Stopping Outbox Relay Worker...")
				return
			case <-ticker.C:
				r.processOutboxEvents()
			}
		}
	}()
}

func (r *OutboxRelay) Stop() {
	close(r.stopCh)
}

func (r *OutboxRelay) processOutboxEvents() {
	var events []*domain.OutboxEvent

	// Fetch unprocessed events
	events, err := r.repo.GetUnprocessed(context.Background(), r.batchSize)
	if err != nil {
		logger.Log.Sugar().Errorf("Failed to fetch outbox events: %v", err)
		return
	}

	if len(events) == 0 {
		return
	}

	// Publish via generic interface
	err = r.publisher.PublishOutboxBatch(context.Background(), r.streamName, events)
	if err != nil {
		logger.Log.Sugar().Errorf("Failed to publish outbox events: %v", err)
		return
	}

	// Mark as processed
	var ids []string
	for _, event := range events {
		ids = append(ids, event.ID)
	}

	err = r.repo.MarkProcessed(context.Background(), ids)
	if err != nil {
		logger.Log.Sugar().Errorf("Failed to mark outbox events as processed (DB may be out of sync): %v", err)
	} else {
		logger.Log.Sugar().Infof("Successfully published %d outbox events to %s", len(events), r.streamName)
	}
}
