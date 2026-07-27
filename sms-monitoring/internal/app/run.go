package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sms-monitoring/internal/domain"
	"sms-monitoring/internal/infrastructure/database"
	"sms-monitoring/internal/infrastructure/logger"
	"sms-monitoring/internal/infrastructure/messagebroker"
	infraRedis "sms-monitoring/internal/infrastructure/redis"
	"sms-monitoring/internal/worker/consumers"
	"sms-monitoring/internal/worker/sweepers"

	"github.com/redis/go-redis/v9"
)

func (a *App) Run() error {
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
	ticker := time.NewTicker(a.cfg.WorkerTickInterval)
	defer ticker.Stop()

	logger.Log.Sugar().Infof("Monitoring Worker started. Scanning every %s\n", a.cfg.WorkerTickInterval)

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

	agentSweeper := sweepers.NewAgentSweeper(a.RedisClient, a.monService, a.cfg.SweeperInterval, int64(a.cfg.AgentPushTTL.Seconds()))
	go agentSweeper.Start(ctx)

	// Start Heartbeat Consumer
	heartbeatConsumer := consumers.NewHeartbeatConsumer(subscriber, a.monService, a.RedisClient)
	go heartbeatConsumer.Start(ctx)

	<-sigCh
	cancel()

	a.Shutdown(context.Background())

	return nil
}

func (a *App) Shutdown(ctx context.Context) {
	logger.Log.Sugar().Info("Shutting down Monitoring Worker...")

	// Wait for running workers to finish
	time.Sleep(2 * time.Second)

	if a.esLogger != nil {
		a.esLogger.Shutdown()
	}

	logger.Log.Sugar().Info("Monitoring Worker stopped.")
}

func (a *App) runProducerCycle(ctx context.Context) {
	lockKey := a.cfg.ProducerLockKey
	tickInterval := a.cfg.WorkerTickInterval

	// Set lock expiration to slightly less than tick interval to tightly block other workers
	// but still allow the lock to expire before the next legitimate cycle.
	lockExpiration := tickInterval - 2*time.Second
	if lockExpiration <= 0 {
		lockExpiration = tickInterval
	}

	// Producer Election
	acquired, _ := database.AcquireLock(ctx, a.RedisClient, lockKey, lockExpiration)
	if acquired {
		queueLen, err := a.RedisClient.LLen(ctx, infraRedis.MonitoringQueueKey).Result()
		if err == nil && queueLen > 0 {
			logger.Log.Sugar().Warnf("[Producer] Queue still has %d items! Skipping push to avoid snowballing.", queueLen)
		} else {
			logger.Log.Sugar().Info("[Producer] Lock acquired. Populating work queue...")
			// Fetch all Server IDs
			serverIDs, err := a.RedisClient.SMembers(ctx, infraRedis.ServerAllIDsKey).Result()
			if err == nil && len(serverIDs) > 0 {
				// Use pipeline to fetch health_check_method for all servers
				pipe := a.RedisClient.Pipeline()
				cmds := make(map[string]*redis.StringCmd)
				for _, id := range serverIDs {
					redisKey := fmt.Sprintf(infraRedis.ServerInfoKeyFmt, id)
					cmds[id] = pipe.HGet(ctx, redisKey, infraRedis.ServerInfoFieldHealthCheckMethod)
				}
				_, _ = pipe.Exec(ctx)

				// Filter out AGENT_PUSH servers
				var pollingServers []interface{}
				for id, cmd := range cmds {
					method, err := cmd.Result()
					if err == nil && method != string(domain.HealthCheckMethodAgentPush) {
						pollingServers = append(pollingServers, id)
					}
				}

				if len(pollingServers) > 0 {
					a.RedisClient.RPush(ctx, infraRedis.MonitoringQueueKey, pollingServers...)
					logger.Log.Sugar().Infof("[Producer] Pushed %d servers to the queue (Ignored %d AGENT_PUSH servers).", len(pollingServers), len(serverIDs)-len(pollingServers))

					// Track duration for this batch
					start := time.Now()
					go func(batchSize int) {
						for {
							time.Sleep(1 * time.Second)
							length, err := a.RedisClient.LLen(context.Background(), infraRedis.MonitoringQueueKey).Result()
							if err != nil || length == 0 {
								duration := time.Since(start)
								logger.Log.Sugar().Infof("[Consumer] Batch of %d servers processed (Duration: %s)", batchSize, duration)
								return
							}
						}
					}(len(pollingServers))
				} else {
					logger.Log.Sugar().Info("[Producer] No polling servers to push.")
				}
			}
		}
	}
}
