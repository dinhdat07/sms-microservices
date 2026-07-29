package redis

const (
	ServerAllIDsKey       = "server:all_ids"
	ServerInfoKeyFmt      = "server:info:%s"
	AgentHeartbeatZSetKey = "monitoring:agent:heartbeats"
	MonitoringQueueKey    = "monitoring:queue"

	// Redis Hash Fields for Server Info
	ServerInfoFieldIPv4              = "ipv4"
	ServerInfoFieldStatus            = "status"
	ServerInfoFieldHealthCheckMethod = "health_check_method"
	ServerInfoFieldSSHPort           = "ssh_port"
	ServerInfoFieldSSHUser           = "ssh_user"
	ServerInfoFieldSSHKey            = "ssh_key"
	ServerInfoFieldAgentEndpoint     = "agent_endpoint"
)
