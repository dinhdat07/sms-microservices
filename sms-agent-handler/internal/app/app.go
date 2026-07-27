package app

import (
	"context"
	"fmt"
	"net/http"

	"sms-agent-handler/internal/config"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type App struct {
	cfg         *config.Config
	RedisClient redis.UniversalClient
	logger      *zap.Logger
	httpServer  *http.Server
}

func NewApp() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	logger, _ := zap.NewProduction()

	opts, err := redis.ParseURL(cfg.RedisURI)
	if err != nil {
		return nil, fmt.Errorf("invalid redis uri: %w", err)
	}

	redisClient := redis.NewClient(opts)

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &App{
		cfg:         cfg,
		RedisClient: redisClient,
		logger:      logger,
	}, nil
}
