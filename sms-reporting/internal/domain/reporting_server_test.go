package domain

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestReportingServer_TableName(t *testing.T) {
	s := &ReportingServer{}
	assert.Equal(t, "reporting_schema.reporting_servers", s.TableName())
}
