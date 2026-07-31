package domain

import "time"

// ReportingServer is a local replica of server data, populated via Events from sms-management.
// It exists so Reporting can generate reports autonomously without calling gRPC to Management.
type ReportingServer struct {
	ServerID  string    `gorm:"type:varchar(255);primaryKey"`
	Name      string    `gorm:"type:varchar(255);uniqueIndex"`
	IPv4      string    `gorm:"type:varchar(45);uniqueIndex"`
	Status    string    `gorm:"type:varchar(50);default:'UNKNOWN'"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (ReportingServer) TableName() string {
	return "reporting_schema.reporting_servers"
}
