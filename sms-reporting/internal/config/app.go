package config

import (
	"strings"
)

type LoggerConfig struct {
	Level         string
	Format        string
	LogMaxSize    int // megabytes
	LogMaxBackups int
	LogMaxAge     int // days
	LogCompress   bool
}

type ReportingConfig struct {
	WorkerCount        int
	JobQueueSize       int
	AdminEmail         string
	CronSpec                 string
	NotificationStream       string
	NotificationStreamMaxLen int64
}

type ConsumerConfig struct {
	Name               string
	ServerStream       string
	ServerGroup        string
	ServerStatusStream string
	ServerStatusGroup  string
}

type Config struct {
	GRPCPort string
	HTTPPort string

	DBUrl string
	Env   string

	Logger    LoggerConfig
	Reporting ReportingConfig
	Consumer  ConsumerConfig
}

func Load() (*Config, error) {
	// load .env into os env
	loadEnv()

	httpPort := strings.TrimSpace(GetEnvDefault("HTTP_PORT", ""))
	if httpPort == "" {
		httpPort = GetEnvDefault("PORT", "8002")
	}

	workerCount, err := GetEnvInt("REPORTING_WORKER_COUNT", 5)
	if err != nil {
		workerCount = 5
	}

	jobQueueSize, err := GetEnvInt("REPORTING_JOB_QUEUE_SIZE", 100)
	if err != nil {
		jobQueueSize = 100
	}

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

	notificationMaxLen, err := GetEnvInt("NOTIFICATION_STREAM_MAXLEN", 100000)
	if err != nil {
		notificationMaxLen = 100000
	}

	cfg := &Config{
		GRPCPort: GetEnvDefault("GRPC_PORT", "50053"),
		HTTPPort: httpPort,
		DBUrl:    GetEnvDefault("DB_URL", ""),
		Env:      GetEnvDefault("ENV", "development"),
		Logger: LoggerConfig{
			Level:         GetEnvDefault("LOG_LEVEL", "info"),
			Format:        GetEnvDefault("LOG_FORMAT", "text"),
			LogMaxSize:    logMaxSize,
			LogMaxBackups: logMaxBackups,
			LogMaxAge:     logMaxAge,
			LogCompress:   logCompress,
		},
		Reporting: ReportingConfig{
			WorkerCount:              workerCount,
			JobQueueSize:             jobQueueSize,
			AdminEmail:               GetEnvDefault("ADMIN_EMAIL", "admin@sms.com"),
			CronSpec:                 GetEnvDefault("SCHEDULER_CRON_SPEC", "0 0 * * *"),
			NotificationStream:       GetEnvDefault("NOTIFICATION_STREAM", "notification_events"),
			NotificationStreamMaxLen: int64(notificationMaxLen),
		},
		Consumer: ConsumerConfig{
			Name:               GetEnvDefault("CONSUMER_NAME", "reporting_worker"),
			ServerStream:       GetEnvDefault("CONSUMER_SERVER_STREAM", "sms.events.server"),
			ServerGroup:        GetEnvDefault("CONSUMER_SERVER_GROUP", "reporting_server_group"),
			ServerStatusStream: GetEnvDefault("CONSUMER_STATUS_STREAM", "sms.events.server_status"),
			ServerStatusGroup:  GetEnvDefault("CONSUMER_STATUS_GROUP", "reporting_status_group"),
		},
	}

	return cfg, nil
}
