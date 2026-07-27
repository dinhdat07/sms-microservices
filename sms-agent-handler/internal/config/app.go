package config

type Config struct {
	Env      string
	Port     string
	RedisURI string
}

func Load() (*Config, error) {
	loadEnv()

	cfg := &Config{
		Env:      GetEnvDefault("ENV", "development"),
		Port:     GetEnvDefault("PORT", "8084"),
		RedisURI: GetEnvDefault("REDIS_URI", "redis://localhost:6379/0"),
	}

	return cfg, nil
}
