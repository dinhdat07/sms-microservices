package domain

import (
	"time"

	"github.com/google/uuid"
)

// DailyUptimeStat represents the aggregated ping observations for a single day across the fleet.
type DailyUptimeStat struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey"`
	Date               time.Time `gorm:"type:date;not null;uniqueIndex"`
	TotalPingCount     int64     `gorm:"not null"`
	SuccessPingCount   int64     `gorm:"not null"`
	CreatedAt          time.Time `gorm:"autoCreateTime"`
	UpdatedAt          time.Time `gorm:"autoUpdateTime"`
}

func (DailyUptimeStat) TableName() string {
	return "reporting_schema.daily_uptime_stats"
}
