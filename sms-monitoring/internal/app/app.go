package app

import (
	"context"
	"fmt"

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

	// Ensure Redis pool size is large enough to handle all BLPOP blocking connections
	if redisCfg != nil && redisCfg.PoolSize < cfg.WorkerConcurrency+50 {
		redisCfg.PoolSize = cfg.WorkerConcurrency + 50
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
	if err := elasticsearch.InitILMAndDataStream(context.Background(), esClient, esCfg.ServerIndex, esCfg.RetentionDays); err != nil {
		logger.Log.Sugar().Errorf("Failed to initialize ES ILM and Data Stream: %v", err)
	}

	// Initialize Dependencies

	stateStore := impl.NewRedisServerStateStore(redisClient)
	publisher := messagebroker.NewRedisPublisher(redisClient, cfg.Publisher.MaxLen)
	monService := service.NewMonitoringService(publisher, stateStore, esLogger, cfg.FailureThreshold)

	logger.Log.Info(fmt.Sprintf("Monitoring Worker started. Scanning every %s with failure threshold %d", cfg.WorkerTickInterval.String(), cfg.FailureThreshold))

	timeouts := checkers.CheckerTimeouts{
		ICMP:      cfg.ICMPTimeout,
		SSH:       cfg.SSHTimeout,
		AgentPull: cfg.AgentPullTimeout,
	}
	factory := checkers.NewHealthCheckerFactory(redisClient, cfg.ICMPPrivileged, timeouts)
	pool := worker.NewWorkerPool(redisClient, monService, factory, cfg.WorkerConcurrency, cfg.WorkerPingTimeout)

	return &App{
		cfg:         cfg,
		RedisClient: redisClient,
		Pool:        pool,
		monService:  monService,
		esLogger:    esLogger,
	}, nil
}
