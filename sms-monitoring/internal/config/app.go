package config

import "time"

type LoggerConfig struct {
	Level         string
	Format        string
	LogMaxSize    int // megabytes
	LogMaxBackups int
	LogMaxAge     int // days
	LogCompress   bool
}

type PublisherConfig struct {
	MaxLen int64
}

type Config struct {
	Env       string
	Logger    LoggerConfig
	Publisher PublisherConfig

	// Worker Configs
	WorkerTickInterval time.Duration
	WorkerConcurrency  int
	WorkerPingTimeout  time.Duration
	FailureThreshold   int
	ProducerLockKey    string

	// Timeout Configs
	ICMPTimeout      time.Duration
	SSHTimeout       time.Duration
	AgentPullTimeout time.Duration

	// ICMP Config
	ICMPPrivileged bool

	// Agent Push Configs
	AgentPushTTL    time.Duration
	AgentPort       string
	SweeperInterval time.Duration
}

func Load() (*Config, error) {
	// load .env into os env
	loadEnv()

	logMaxSize, err := GetEnvInt("LOG_MAX_SIZE", 10)
	if err != nil {
		logMaxSize = 10
	}

	logMaxBackups, err := GetEnvInt("LOG_MAX_BACKUPS", 3)
	if err != nil {
		logMaxBackups = 3
	}

	logMaxAge, err := GetEnvInt("LOG_MAX_AGE", 28)
	if err != nil {
		logMaxAge = 28
	}

	logCompress, err := GetEnvBool("LOG_COMPRESS", true)
	if err != nil {
		logCompress = true
	}

	streamMaxLen, err := GetEnvInt("STREAM_MAXLEN", 1000000)
	if err != nil {
		streamMaxLen = 1000000
	}

	workerTickInterval, _ := GetEnvDuration("MONITORING_WORKER_TICK_INTERVAL", 60*time.Second)
	workerConcurrency, _ := GetEnvInt("MONITORING_WORKER_CONCURRENCY", 100)
	workerPingTimeout, _ := GetEnvDuration("MONITORING_WORKER_PING_TIMEOUT", 3*time.Second)
	failureThreshold, _ := GetEnvInt("MONITORING_FAILURE_THRESHOLD", 1)

	icmpTimeout, _ := GetEnvDuration("MONITORING_ICMP_TIMEOUT", 3*time.Second)
	sshTimeout, _ := GetEnvDuration("MONITORING_SSH_TIMEOUT", 10*time.Second)
	agentPullTimeout, _ := GetEnvDuration("MONITORING_AGENT_PULL_TIMEOUT", 10*time.Second)

	icmpPrivileged, _ := GetEnvBool("ICMP_PRIVILEGED", false)

	agentPushTTLSecs, _ := GetEnvInt("MONITORING_AGENT_PUSH_TTL", 60)
	sweeperInterval, _ := GetEnvDuration("MONITORING_SWEEPER_INTERVAL", 10*time.Second)

	cfg := &Config{
		Env: GetEnvDefault("ENV", "development"),
		Logger: LoggerConfig{
			Level:         GetEnvDefault("LOG_LEVEL", "info"),
			Format:        GetEnvDefault("LOG_FORMAT", "text"),
			LogMaxSize:    logMaxSize,
			LogMaxBackups: logMaxBackups,
			LogMaxAge:     logMaxAge,
			LogCompress:   logCompress,
		},
		Publisher: PublisherConfig{
			MaxLen: int64(streamMaxLen),
		},
		WorkerTickInterval: workerTickInterval,
		WorkerConcurrency:  workerConcurrency,
		WorkerPingTimeout:  workerPingTimeout,
		FailureThreshold:   failureThreshold,
		ProducerLockKey:    GetEnvDefault("MONITORING_PRODUCER_LOCK_KEY", "lock:monitoring_producer"),
		ICMPTimeout:        icmpTimeout,
		SSHTimeout:         sshTimeout,
		AgentPullTimeout:   agentPullTimeout,
		ICMPPrivileged:     icmpPrivileged,
		AgentPushTTL:       time.Duration(agentPushTTLSecs) * time.Second,
		AgentPort:          GetEnvDefault("MONITORING_AGENT_PORT", "8084"), // Changed default to 8084 to avoid conflict
		SweeperInterval:    sweeperInterval,
	}

	return cfg, nil
}
