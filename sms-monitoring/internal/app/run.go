package app

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sms-monitoring/internal/config"
	"sms-monitoring/internal/handler/rest"
	"sms-monitoring/internal/infrastructure/database"
	"sms-monitoring/internal/infrastructure/logger"
	"sms-monitoring/internal/infrastructure/messagebroker"
	"sms-monitoring/internal/worker/consumers"
	"sms-monitoring/internal/worker/sweepers"
)

func (a *App) Run() error {
	tickInterval, _ := config.GetEnvDuration("MONITORING_WORKER_TICK_INTERVAL", 60*time.Second)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Start Stream Consumer
	subscriber := messagebroker.NewRedisSubscriber(a.RedisClient)
	streamConsumer := consumers.NewStreamConsumer(subscriber, a.RedisClient)
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

	sweeperInterval, _ := config.GetEnvDuration("MONITORING_SWEEPER_INTERVAL", 10*time.Second)
	pushTTL, _ := config.GetEnvInt("MONITORING_AGENT_PUSH_TTL", 60)

	// Start Agent Push Sweeper
	agentSweeper := sweepers.NewAgentSweeper(a.RedisClient, a.monService, sweeperInterval, int64(pushTTL))
	go agentSweeper.Start(ctx)

	// Start HTTP Server for Agent Push
	agentHandler := rest.NewAgentHandler(a.RedisClient)
	mux := http.NewServeMux()
	mux.Handle("/api/v1/agent/heartbeat", agentHandler)
	httpServer := &http.Server{
		Addr:    ":8080", // Could be configurable
		Handler: mux,
	}
	go func() {
		logger.Log.Sugar().Info("Agent Push Server listening on :8080")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Sugar().Errorf("HTTP Server error: %v", err)
		}
	}()

	<-sigCh
	logger.Log.Sugar().Info("Shutting down Monitoring Worker...")
	cancel()
	httpServer.Shutdown(context.Background())

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
	tickInterval, _ := config.GetEnvDuration("MONITORING_WORKER_TICK_INTERVAL", 60*time.Second)

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
