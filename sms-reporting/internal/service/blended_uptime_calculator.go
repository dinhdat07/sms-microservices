package service

import (
	"context"
	"time"

	"sms-reporting/internal/domain"
	"sms-reporting/internal/repository"
)

type BlendedUptimeCalculator struct {
	repo        repository.ReportingRepository
	rawProvider domain.RawUptimeProvider
}

func NewBlendedUptimeCalculator(repo repository.ReportingRepository, rawProvider domain.RawUptimeProvider) domain.UptimeCalculator {
	return &BlendedUptimeCalculator{
		repo:        repo,
		rawProvider: rawProvider,
	}
}

func (c *BlendedUptimeCalculator) calculateRawUptimeStats(ctx context.Context, startTime, endTime time.Time) (int64, int64, error) {
	stats, err := c.repo.GetDailyUptimeStats(ctx, startTime, endTime)
	if err != nil {
		return 0, 0, err
	}

	var totalSuccess int64
	var totalCount int64

	// Track which days are already covered by DB
	coveredDays := make(map[string]bool)
	for _, stat := range stats {
		totalSuccess += stat.SuccessPingCount
		totalCount += stat.TotalPingCount
		coveredDays[stat.Date.Format("2006-01-02")] = true
	}

	curr := startTime
	for !curr.After(endTime) {
		dateStr := curr.Format("2006-01-02")
		if !coveredDays[dateStr] {
			dayStart := time.Date(curr.Year(), curr.Month(), curr.Day(), 0, 0, 0, 0, curr.Location())
			dayEnd := dayStart.Add(24 * time.Hour).Add(-time.Nanosecond)

			if dayStart.Before(startTime) {
				dayStart = startTime
			}
			if dayEnd.After(endTime) {
				dayEnd = endTime
			}

			success, total, err := c.rawProvider.CalculateRawUptimeStats(ctx, dayStart, dayEnd)
			if err != nil {
				return 0, 0, err
			}
			totalSuccess += success
			totalCount += total
		}
		curr = curr.Add(24 * time.Hour)
	}

	return totalSuccess, totalCount, nil
}

func (c *BlendedUptimeCalculator) CalculateUptime(ctx context.Context, startTime, endTime time.Time) (float64, error) {
	successCount, totalCount, err := c.calculateRawUptimeStats(ctx, startTime, endTime)
	if err != nil {
		return 0, err
	}
	if totalCount == 0 {
		return 0, nil
	}
	return (float64(successCount) / float64(totalCount)) * 100, nil
}
