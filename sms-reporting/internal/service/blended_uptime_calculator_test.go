package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"sms-reporting/internal/domain"
	mockDomain "sms-reporting/internal/domain/mock"
	mockRepo "sms-reporting/internal/repository/mock"
	"github.com/stretchr/testify/mock"
)

func TestBlendedUptimeCalculator_CalculateUptime(t *testing.T) {
	repo := mockRepo.NewMockReportingRepository(t)
	rawProvider := mockDomain.NewMockRawUptimeProvider(t)
	blendedCalc := NewBlendedUptimeCalculator(repo, rawProvider)

	ctx := context.Background()
	now := time.Now()
	startTime := time.Date(now.Year(), now.Month(), now.Day()-2, 0, 0, 0, 0, now.Location())
	endTime := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())

	// Mock DB return for 1 day
	stats := []domain.DailyUptimeStat{
		{Date: startTime, SuccessPingCount: 50, TotalPingCount: 100},
	}
	repo.On("GetDailyUptimeStats", ctx, startTime, endTime).Return(stats, nil)

	// Mock ES return for missing days
	rawProvider.On("CalculateRawUptimeStats", ctx, mock.Anything, mock.Anything).Return(int64(950), int64(900), nil)

	uptime, err := blendedCalc.CalculateUptime(ctx, startTime, endTime)
	assert.NoError(t, err)
	// (50 + 950 + 950) / (100 + 900 + 900) = 1950 / 1900 = 102.63%
	assert.Greater(t, uptime, float64(100))
}

func TestBlendedUptimeCalculator_TotalCountZero(t *testing.T) {
	repo := mockRepo.NewMockReportingRepository(t)
	rawProvider := mockDomain.NewMockRawUptimeProvider(t)
	blendedCalc := NewBlendedUptimeCalculator(repo, rawProvider)

	ctx := context.Background()
	startTime := time.Now()
	endTime := startTime

	repo.On("GetDailyUptimeStats", ctx, startTime, endTime).Return([]domain.DailyUptimeStat{}, nil)
	rawProvider.On("CalculateRawUptimeStats", ctx, mock.Anything, mock.Anything).Return(int64(0), int64(0), nil)

	uptime, err := blendedCalc.CalculateUptime(ctx, startTime, endTime)
	assert.NoError(t, err)
	assert.Equal(t, float64(0), uptime)
}
