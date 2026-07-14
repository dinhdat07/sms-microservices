package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidEmail = errors.New("invalid email address")

const (
	ReportStatusPending    = "PENDING"
	ReportStatusProcessing = "PROCESSING"
	ReportStatusCompleted  = "COMPLETED"
	ReportStatusFailed     = "FAILED"
)

// ReportRequest represents an asynchronous request to generate a server uptime report.
type ReportRequest struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	RequestorEmail string    `gorm:"type:varchar(255);not null"`
	StartTime      time.Time `gorm:"not null"`
	EndTime        time.Time `gorm:"not null"`
	Status         string    `gorm:"type:varchar(50);not null"`
	CorrelationID  string    `gorm:"type:varchar(255);not null;index"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
}

func (ReportRequest) TableName() string {
	return "reporting_schema.report_requests"
}

// NewReportRequest is a factory function to create a new report request
func NewReportRequest(requestorEmail string, startTime, endTime time.Time, correlationID string) (*ReportRequest, error) {
	if strings.TrimSpace(requestorEmail) == "" {
		return nil, ErrInvalidEmail
	}

	requestID := uuid.New()

	req := &ReportRequest{
		ID:             requestID,
		RequestorEmail: requestorEmail,
		StartTime:      startTime,
		EndTime:        endTime,
		Status:         ReportStatusPending,
		CorrelationID:  correlationID,
	}

	return req, nil
}
