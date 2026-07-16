package config

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

var loadEnvOnce sync.Once

func loadEnv() {
	loadEnvOnce.Do(func() {
		envFile := os.Getenv("ENV_FILE")
		if envFile != "" {
			_ = godotenv.Load(envFile)
			return
		}

		_ = godotenv.Load()
	})
}

func getEnv(key string) string {
	fileKey := key + "_FILE"
	if filePath := os.Getenv(fileKey); filePath != "" {
		if content, err := os.ReadFile(filePath); err == nil {
			return strings.TrimSpace(string(content))
		}
	}
	return os.Getenv(key)
}

func GetEnvDefault(key, fallback string) string {
	val := getEnv(key)
	if val == "" {
		return fallback
	}
	return val
}

func GetEnvBool(key string, fallback bool) (bool, error) {
	val := getEnv(key)
	if val == "" {
		return fallback, nil
	}
	return strconv.ParseBool(val)
}

func GetEnvInt(key string, fallback int) (int, error) {
	val := getEnv(key)
	if val == "" {
		return fallback, nil
	}
	return strconv.Atoi(val)
}

func GetEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	val := getEnv(key)
	if val == "" {
		return fallback, nil
	}
	return time.ParseDuration(val)
}

func GetEnvFloat(key string, fallback float64) (float64, error) {
	val := os.Getenv(key)
	if val == "" {
		return fallback, nil
	}
	return strconv.ParseFloat(val, 64)
}