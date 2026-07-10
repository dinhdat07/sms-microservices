package domain

import (
	"time"

	"github.com/google/uuid"
)

type AuthSession struct {
	ID         uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID     uint       `gorm:"index"`
	ExpiresAt  time.Time  `gorm:"not null"`
	LastUsedAt *time.Time
	RevokedAt  *time.Time
	IPAddress  string     `gorm:"type:varchar(45)"`
	UserAgent  string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}