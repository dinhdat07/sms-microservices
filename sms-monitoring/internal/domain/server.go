package domain

type ServerStatus string

const (
	ServerStatusOnline  ServerStatus = "ONLINE"
	ServerStatusOffline ServerStatus = "OFFLINE"
	ServerStatusUnknown ServerStatus = "UNKNOWN"
)

type HealthCheckMethod string

const (
	HealthCheckMethodICMP      HealthCheckMethod = "ICMP"
	HealthCheckMethodSSH       HealthCheckMethod = "SSH"
	HealthCheckMethodAgentPull HealthCheckMethod = "AGENT_PULL"
	HealthCheckMethodAgentPush HealthCheckMethod = "AGENT_PUSH"
)
