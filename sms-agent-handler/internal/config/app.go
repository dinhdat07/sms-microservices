package config

type Config struct {
	Env       string
	Port      string
	RedisURI  string
	MasterKey string
}

func Load() (*Config, error) {
	loadEnv()

	cfg := &Config{
		Env:       GetEnvDefault("ENV", "development"),
		Port:      GetEnvDefault("PORT", "8084"),
		RedisURI:  GetEnvDefault("REDIS_URI", "redis://localhost:6379/0"),
		MasterKey: GetEnvDefault("MASTER_KEY", "0123456789abcdef0123456789abcdef"),
	}

	return cfg, nil
}
