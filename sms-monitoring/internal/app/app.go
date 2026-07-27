package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

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

	// Settings
	concurrency, _ := config.GetEnvInt("MONITORING_WORKER_CONCURRENCY", 100)
	pingTimeout, _ := config.GetEnvDuration("MONITORING_WORKER_PING_TIMEOUT", 3*time.Second)

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
	threshold, _ := config.GetEnvInt("MONITORING_FAILURE_THRESHOLD", 1)
	publisher := messagebroker.NewRedisPublisher(redisClient, cfg.Publisher.MaxLen)
	monService := service.NewMonitoringService(publisher, stateStore, esLogger, threshold)

	tickInterval, _ := config.GetEnvDuration("MONITORING_WORKER_TICK_INTERVAL", 30*time.Second)
	logger.Log.Info(fmt.Sprintf("Monitoring Worker started. Scanning every %s with failure threshold %d", tickInterval.String(), threshold))

	// Unprivileged ping for non-root environments (Set to true if running as root on Linux)
	privilegedStr := os.Getenv("ICMP_PRIVILEGED")
	privileged, _ := strconv.ParseBool(privilegedStr)

	icmpTimeout, _ := config.GetEnvDuration("MONITORING_ICMP_TIMEOUT", 3*time.Second)
	sshTimeout, _ := config.GetEnvDuration("MONITORING_SSH_TIMEOUT", 10*time.Second)
	agentPullTimeout, _ := config.GetEnvDuration("MONITORING_AGENT_PULL_TIMEOUT", 10*time.Second)

	timeouts := checkers.CheckerTimeouts{
		ICMP:      icmpTimeout,
		SSH:       sshTimeout,
		AgentPull: agentPullTimeout,
	}
	factory := checkers.NewHealthCheckerFactory(redisClient, privileged, timeouts)
	pool := worker.NewWorkerPool(redisClient, monService, factory, concurrency, pingTimeout)

	return &App{
		RedisClient: redisClient,
		Pool:        pool,
		monService:  monService,
		esLogger:    esLogger,
	}, nil
}
