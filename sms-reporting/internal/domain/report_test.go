package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestReportRequest_TableName(t *testing.T) {
	r := &ReportRequest{}
	assert.Equal(t, "reporting_schema.report_requests", r.TableName())
}

func TestNewReportRequest(t *testing.T) {
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()
	
	r, err := NewReportRequest("test@example.com", start, end, "corr-1")
	assert.NoError(t, err)
	assert.NotNil(t, r)
	assert.Equal(t, "test@example.com", r.RequestorEmail)
	assert.Equal(t, ReportStatusPending, r.Status)
	assert.Equal(t, "corr-1", r.CorrelationID)
	assert.NotEqual(t, uuid.Nil, r.ID)
}

func TestNewReportRequest_InvalidEmail(t *testing.T) {
	_, err := NewReportRequest("", time.Now(), time.Now(), "corr")
	assert.Error(t, err)
}
