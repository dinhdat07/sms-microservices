package config

type LoggerConfig struct {
	Level         string
	Format        string
	LogMaxSize    int // megabytes
	LogMaxBackups int
	LogMaxAge     int // days
	LogCompress   bool
}

type Config struct {
	Env    string
	Logger LoggerConfig
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
	}

	return cfg, nil
}

