package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"sms-monitoring/internal/config"
	"sms-monitoring/internal/infrastructure/database"
	"sms-monitoring/internal/infrastructure/elasticsearch"
	"sms-monitoring/internal/infrastructure/logger"
	"sms-monitoring/internal/infrastructure/messagebroker"
	"sms-monitoring/internal/repository/impl"
	"sms-monitoring/internal/service"
	"sms-monitoring/internal/worker"

	"github.com/redis/go-redis/v9"
)

type App struct {
	RedisClient redis.UniversalClient
	Pool        worker.Pool
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
	threshold, _ := config.GetEnvInt("MONITORING_FAILURE_THRESHOLD", 2)
	publisher := messagebroker.NewRedisPublisher(redisClient, cfg.Publisher.MaxLen)
	monService := service.NewMonitoringService(publisher, stateStore, esLogger, threshold)

	tickInterval, _ := config.GetEnvDuration("MONITORING_WORKER_TICK_INTERVAL", 30*time.Second)
	logger.Log.Info(fmt.Sprintf("Monitoring Worker started. Scanning every %s with failure threshold %d", tickInterval.String(), threshold))

	// Unprivileged ping for non-root environments (Set to true if running as root on Linux)
	privilegedStr := os.Getenv("ICMP_PRIVILEGED")
	privileged, _ := strconv.ParseBool(privilegedStr)
	pinger := worker.NewICMPPinger(privileged)

	pool := worker.NewWorkerPool(redisClient, monService, pinger, concurrency, pingTimeout)

	return &App{
		RedisClient: redisClient,
		Pool:        pool,
		esLogger:    esLogger,
	}, nil
}

func (a *App) Shutdown() {
	if a.esLogger != nil {
		a.esLogger.Shutdown()
	}
}

func (a *App) Run() error {
	tickInterval, _ := config.GetEnvDuration("MONITORING_WORKER_TICK_INTERVAL", 30*time.Second)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Start Stream Consumer
	subscriber := messagebroker.NewRedisSubscriber(a.RedisClient)
	streamConsumer := worker.NewStreamConsumer(subscriber, a.RedisClient)
	go streamConsumer.Start(ctx)

	// Start Scheduler
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	logger.Log.Sugar().Infof("Monitoring Worker started. Scanning every %s\n", tickInterval)

	// Always-on Consumer
	go func() {
		err := a.Pool.Run(ctx)
		if err != nil && ctx.Err() == nil {
			logger.Log.Sugar().Errorf("[Consumer] Worker pool stopped with error: %v", err)
		} else {
			logger.Log.Sugar().Info("[Consumer] Worker pool stopped")
		}
	}()

	// Ticker-based Producer
	go func() {
		a.runProducerCycle(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.runProducerCycle(ctx)
			}
		}
	}()

	<-sigCh
	logger.Log.Sugar().Info("Shutting down Monitoring Worker...")
	cancel()

	// Wait for running workers to finish
	time.Sleep(2 * time.Second)

	if a.esLogger != nil {
		a.esLogger.Shutdown()
	}

	logger.Log.Sugar().Info("Monitoring Worker stopped.")
	return nil
}

func (a *App) runProducerCycle(ctx context.Context) {
	lockKey := config.GetEnvDefault("MONITORING_PRODUCER_LOCK_KEY", "lock:monitoring_producer")
	tickInterval, _ := config.GetEnvDuration("MONITORING_WORKER_TICK_INTERVAL", 30*time.Second)
	
	// Set lock expiration to slightly less than tick interval to tightly block other workers
	// but still allow the lock to expire before the next legitimate cycle.
	lockExpiration := tickInterval - 2*time.Second
	if lockExpiration <= 0 {
		lockExpiration = tickInterval
	}

	// Producer Election
	acquired, _ := database.AcquireLock(ctx, a.RedisClient, lockKey, lockExpiration)
	if acquired {
		queueLen, err := a.RedisClient.LLen(ctx, "monitoring:queue").Result()
		if err == nil && queueLen > 0 {
			logger.Log.Sugar().Warnf("[Producer] Queue still has %d items! Skipping push to avoid snowballing.", queueLen)
		} else {
			logger.Log.Sugar().Info("[Producer] Lock acquired. Populating work queue...")
			// Fetch all Server IDs
			serverIDs, err := a.RedisClient.SMembers(ctx, "server:all_ids").Result()
			if err == nil && len(serverIDs) > 0 {
				// Push all servers to queue
				args := make([]interface{}, len(serverIDs))
				for i, v := range serverIDs {
					args[i] = v
				}
				a.RedisClient.RPush(ctx, "monitoring:queue", args...)
				logger.Log.Sugar().Infof("[Producer] Pushed %d servers to the queue.", len(serverIDs))

				// Track duration for this batch
				start := time.Now()
				go func(batchSize int) {
					for {
						time.Sleep(1 * time.Second)
						length, err := a.RedisClient.LLen(context.Background(), "monitoring:queue").Result()
						if err != nil || length == 0 {
							duration := time.Since(start)
							logger.Log.Sugar().Infof("[Consumer] Batch of %d servers processed (Duration: %s)", batchSize, duration)
							return
						}
					}
				}(len(serverIDs))
			}
		}
	}
}
