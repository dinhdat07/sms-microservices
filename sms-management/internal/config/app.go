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

type SMTPConfig struct {
	Host     string
	Port     string
	UseAuth  bool
	UseTLS   bool
	Username string
	Password string
	From     string
	FromName string
}

type ReportingConfig struct {
	WorkerCount  int
	JobQueueSize int
}

type OutboxConfig struct {
	StreamName string
	BatchSize  int
	IntervalMs int // in milliseconds
	MaxLen     int64
}

type Config struct {
	GRPCPort string
	HTTPPort string

	EncryptionKey      string
	AgentSigningSecret string
	DBUrl         string
	JWTSecret     string
	Env           string
	AdminEmail    string
	AdminPassword string
	UserEmail     string
	UserPassword  string
	ApiBaseUrl    string
	FrontEndUrl   string

	Logger    LoggerConfig
	SMTP      SMTPConfig
	Reporting ReportingConfig
	Outbox    OutboxConfig
}

func Load() (*Config, error) {
	// load .env into os env
	loadEnv()

	jwtSecret := GetEnvDefault("JWT_SECRET", "")
	if jwtSecret == "" {
		// Backward compatibility with older env key.
		jwtSecret = GetEnvDefault("JWT_SECRET_KEY", "")
	}

	httpPort := strings.TrimSpace(GetEnvDefault("HTTP_PORT", ""))
	if httpPort == "" {
		httpPort = GetEnvDefault("PORT", "8000")
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
	
	outboxBatchSize, err := GetEnvInt("OUTBOX_BATCH_SIZE", 500)
	if err != nil {
		outboxBatchSize = 500
	}
	
	outboxIntervalMs, err := GetEnvInt("OUTBOX_INTERVAL_MS", 2000)
	if err != nil {
		outboxIntervalMs = 2000
	}
	
	outboxMaxLen, err := GetEnvInt("OUTBOX_STREAM_MAXLEN", 1000000)
	if err != nil {
		outboxMaxLen = 1000000
	}

	cfg := &Config{
		GRPCPort:      GetEnvDefault("GRPC_PORT", "50051"),
		HTTPPort:      httpPort,
		EncryptionKey:      GetEnvDefault("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef"),
		AgentSigningSecret: GetEnvDefault("AGENT_SIGNING_SECRET", "super-secret-agent-signing-key"),
		DBUrl:         GetEnvDefault("DB_URL", ""),
		JWTSecret:     jwtSecret,
		Env:           GetEnvDefault("ENV", "development"),
		AdminEmail:    GetEnvDefault("ADMIN_EMAIL", ""),
		AdminPassword: GetEnvDefault("ADMIN_PASSWORD", ""),
		UserEmail:     GetEnvDefault("USER_EMAIL", ""),
		UserPassword:  GetEnvDefault("USER_PASSWORD", ""),
		ApiBaseUrl:    GetEnvDefault("API_BASE_URL", ""),
		FrontEndUrl:   GetEnvDefault("FRONTEND_BASE_URL", ""),
		Logger: LoggerConfig{
			Level:         GetEnvDefault("LOG_LEVEL", "info"),
			Format:        GetEnvDefault("LOG_FORMAT", "text"),
			LogMaxSize:    logMaxSize,
			LogMaxBackups: logMaxBackups,
			LogMaxAge:     logMaxAge,
			LogCompress:   logCompress,
		},
		SMTP: SMTPConfig{
			Host:     GetEnvDefault("SMTP_HOST", "localhost"),
			Port:     GetEnvDefault("SMTP_PORT", "1025"),
			UseAuth:  GetEnvDefault("SMTP_USE_AUTH", "false") == "true",
			UseTLS:   GetEnvDefault("SMTP_USE_TLS", "false") == "true",
			Username: GetEnvDefault("SMTP_USERNAME", ""),
			Password: GetEnvDefault("SMTP_PASSWORD", ""),
			From:     GetEnvDefault("SMTP_FROM", "no-reply@sms.com"),
			FromName: GetEnvDefault("SMTP_FROM_NAME", "SMS Server Management"),
		},
		Reporting: ReportingConfig{
			WorkerCount:  workerCount,
			JobQueueSize: jobQueueSize,
		},
		Outbox: OutboxConfig{
			StreamName: GetEnvDefault("OUTBOX_STREAM_NAME", "sms.events.server"),
			BatchSize:  outboxBatchSize,
			IntervalMs: outboxIntervalMs,
			MaxLen:     int64(outboxMaxLen),
		},
	}

	return cfg, nil
}
