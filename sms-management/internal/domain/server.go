package domain

import (
	"time"
)

type ServerStatus string

const (
	ServerStatusOnline  ServerStatus = "ONLINE"
	ServerStatusOffline ServerStatus = "OFFLINE"
)

func (s ServerStatus) IsValid() bool {
	switch s {
	case ServerStatusOnline, ServerStatusOffline:
		return true
	}
	return false
}

type Server struct {
	ServerID            string       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"server_id"`
	ServerName          string       `gorm:"type:varchar(255);uniqueIndex;not null" json:"server_name"`
	IPv4                string       `gorm:"type:varchar(15);uniqueIndex;not null" json:"ipv4"`
	CurrentStatus       ServerStatus `gorm:"type:varchar(20);default:'ONLINE';not null" json:"current_status"`
	ConsecutiveFailures int          `gorm:"type:int;default:0;not null" json:"consecutive_failures"`
	CreatedAt           time.Time    `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time    `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Server) TableName() string {
	return "management_schema.servers"
}
