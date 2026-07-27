package domain

import (
	"time"
)

type ServerStatus string

const (
	ServerStatusOnline  ServerStatus = "ONLINE"
	ServerStatusOffline ServerStatus = "OFFLINE"
	ServerStatusUnknown ServerStatus = "UNKNOWN"
)

func (s ServerStatus) IsValid() bool {
	switch s {
	case ServerStatusOnline, ServerStatusOffline, ServerStatusUnknown:
		return true
	}
	return false
}

type HealthCheckMethod string

const (
	MethodICMP      HealthCheckMethod = "ICMP"
	MethodSSH       HealthCheckMethod = "SSH"
	MethodAgentPull HealthCheckMethod = "AGENT_PULL"
	MethodAgentPush HealthCheckMethod = "AGENT_PUSH"
)

type Server struct {
	ServerID            string            `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"server_id"`
	ServerName          string            `gorm:"type:varchar(255);uniqueIndex;not null" json:"server_name"`
	IPv4                string            `gorm:"type:varchar(15);uniqueIndex;not null" json:"ipv4"`
	CurrentStatus       ServerStatus      `gorm:"type:varchar(20);default:'UNKNOWN';not null" json:"current_status"`
	HealthCheckMethod   HealthCheckMethod `gorm:"type:varchar(20);default:'ICMP';not null" json:"health_check_method"`
	SSHPort             int               `gorm:"type:int" json:"ssh_port"`
	SSHUser             string            `gorm:"type:varchar(255)" json:"ssh_user"`
	SSHKey              string            `gorm:"type:text" json:"ssh_key"`
	AgentEndpoint       string            `gorm:"type:varchar(255)" json:"agent_endpoint"`
	ConsecutiveFailures int               `gorm:"type:int;default:0;not null" json:"consecutive_failures"`
	CreatedAt           time.Time         `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time         `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Server) TableName() string {
	return "management_schema.servers"
}
