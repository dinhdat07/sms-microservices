package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Env                string
	RedisEnabled       bool
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	NotificationStream string
	NotificationGroup  string
	ConsumerName       string

	SMTPHost     string
	SMTPPort     string
	SMTPUseAuth  bool
	SMTPUseTLS   bool
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
	SMTPFromName string

	LogLevel  string
	LogFormat string
}

func LoadConfig() Config {
	_ = godotenv.Load()

	cfg := Config{
		Env:                getEnv("ENV", "development"),
		RedisEnabled:       getEnv("REDIS_ENABLED", "false") == "true",
		RedisAddr:          getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		RedisDB:            getEnvAsInt("REDIS_DB", 0),
		NotificationStream: getEnv("NOTIFICATION_STREAM", "notification_events"),
		NotificationGroup:  getEnv("NOTIFICATION_GROUP", "notification_workers"),
		ConsumerName:       getEnv("CONSUMER_NAME", "notification_worker_1"),

		SMTPHost:     getEnv("SMTP_HOST", "localhost"),
		SMTPPort:     getEnv("SMTP_PORT", "1025"),
		SMTPUseAuth:  getEnv("SMTP_USE_AUTH", "false") == "true",
		SMTPUseTLS:   getEnv("SMTP_USE_TLS", "false") == "true",
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("SMTP_FROM", "noreply@sms.local"),
		SMTPFromName: getEnv("SMTP_FROM_NAME", "SMS Notification"),

		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "text"),
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsInt(name string, fallback int) int {
	valStr := getEnv(name, "")
	if valStr == "" {
		return fallback
	}
	var val int
	_, err := fmt.Sscanf(valStr, "%d", &val)
	if err != nil {
		return fallback
	}
	return val
}
