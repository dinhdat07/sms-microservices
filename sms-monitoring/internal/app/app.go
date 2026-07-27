package app

import (
	"context"
	"fmt"
	"net/http"

	"sms-monitoring/internal/config"
	"sms-monitoring/internal/infrastructure/database"
	"sms-monitoring/internal/infrastructure/elasticsearch"
	"sms-monitoring/internal/infrastructure/logger"
	"sms-monitoring/internal/infrastructure/messagebroker"
	"sms-monitoring/internal/repository/impl"
	"sms-monitoring/internal/service"
	"sms-monitoring/internal/worker"
	"sms-monitoring/internal/worker/checkers"

	"github.com/redis/go-redis/v9"
)

type App struct {
	cfg         *config.Config
	RedisClient redis.UniversalClient
	Pool        worker.Pool
	monService  service.MonitoringService
	esLogger    elasticsearch.ObservationLogger
	httpServer  *http.Server
}

func NewApp() (*App, error) {
	// Initialize logger
	cfg, _ := config.Load()
	if cfg != nil {
		logger.InitLogger(cfg.Logger, "monitoring-worker")
	} else {
		logger.InitLogger(config.LoggerConfig{}, "monitoring-worker")
	}

	redisCfg, err := config.LoadRedisConfig()
	if err != nil {
		logger.Log.Sugar().Errorf("Failed to load redis config: %v", err)
	}

	// Load workers Configs
	concurrency := cfg.WorkerConcurrency
	pingTimeout := cfg.WorkerPingTimeout

	// Ensure Redis pool size is large enough to handle all BLPOP blocking connections
	if redisCfg != nil && redisCfg.PoolSize < concurrency+50 {
		redisCfg.PoolSize = concurrency + 50
		logger.Log.Sugar().Infof("Adjusted Redis PoolSize to %d to support BLPOP concurrency", redisCfg.PoolSize)
	}

	// Initialize Redis
	redisClient := database.NewRedisClient(redisCfg)
	if redisClient == nil {
		return nil, fmt.Errorf("redis is required for Monitoring Worker")
	}
	if err := database.PingRedis(context.Background(), redisClient); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	// Initialize Elasticsearch
	esCfg := config.LoadElasticsearchConfig()
	esClient, err := database.NewElasticsearchClient(context.Background(), []string{esCfg.URL})
	if err != nil {
		return nil, fmt.Errorf("elasticsearch connection failed: %w", err)
	}
	esLogger := elasticsearch.NewObservationLogger(esClient, esCfg.ServerIndex, config.LoadObservationLoggerConfig())

	// Initialize Dependencies

	stateStore := impl.NewRedisServerStateStore(redisClient)
	threshold := cfg.FailureThreshold
	publisher := messagebroker.NewRedisPublisher(redisClient, cfg.Publisher.MaxLen)
	monService := service.NewMonitoringService(publisher, stateStore, esLogger, threshold)

	tickInterval := cfg.WorkerTickInterval
	logger.Log.Info(fmt.Sprintf("Monitoring Worker started. Scanning every %s with failure threshold %d", tickInterval.String(), threshold))

	timeouts := checkers.CheckerTimeouts{
		ICMP:         cfg.ICMPTimeout,
		SSH:          cfg.SSHTimeout,
		AgentPull:    cfg.AgentPullTimeout,
		AgentPushTTL: cfg.AgentPushTTL,
	}
	factory := checkers.NewHealthCheckerFactory(redisClient, cfg.ICMPPrivileged, timeouts)
	pool := worker.NewWorkerPool(redisClient, monService, factory, concurrency, pingTimeout)

	return &App{
		cfg:         cfg,
		RedisClient: redisClient,
		Pool:        pool,
		monService:  monService,
		esLogger:    esLogger,
	}, nil
}
