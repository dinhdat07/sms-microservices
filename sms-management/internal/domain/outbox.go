package domain

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventServerCreated EventType = "ServerCreated"
	EventServerUpdated EventType = "ServerUpdated"
	EventServerDeleted EventType = "ServerDeleted"
)

type OutboxEvent struct {
	ID            string    `gorm:"primaryKey;type:varchar(36)"`
	AggregateType string    `gorm:"type:varchar(50);not null;index"`
	AggregateID   string    `gorm:"type:varchar(255);not null;index"`
	EventType     EventType `gorm:"type:varchar(50);not null"`
	Payload       []byte    `gorm:"type:jsonb;not null"`
	IsProcessed   bool      `gorm:"not null;default:false;index"`
	CreatedAt     time.Time `gorm:"not null"`
}

func NewOutboxEvent(aggregateType, aggregateID string, eventType EventType, payload []byte) *OutboxEvent {
	return &OutboxEvent{
		ID:            uuid.New().String(),
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		EventType:     eventType,
		Payload:       payload,
		IsProcessed:   false,
		CreatedAt:     time.Now().UTC(),
	}
}
