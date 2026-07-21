package worker

import (
	"context"
	"fmt"
	"sms-monitoring/internal/infrastructure/logger"
	"sync"
	"time"

	infraRedis "sms-monitoring/internal/infrastructure/redis"
	"sms-monitoring/internal/service"

	"github.com/redis/go-redis/v9"
)

type Pool interface {
	Run(ctx context.Context) error
}

type workerPool struct {
	rdb         redis.UniversalClient
	monService  service.MonitoringService
	pinger      Pinger
	concurrency int
	timeout     time.Duration
}

func NewWorkerPool(rdb redis.UniversalClient, monService service.MonitoringService, pinger Pinger, concurrency int, timeout time.Duration) Pool {
	return &workerPool{
		rdb:         rdb,
		monService:  monService,
		pinger:      pinger,
		concurrency: concurrency,
		timeout:     timeout,
	}
}

func (w *workerPool) Run(ctx context.Context) error {
	var wg sync.WaitGroup

	logger.Log.Sugar().Infof("Starting Worker Pool with concurrency %d", w.concurrency)

	// Spawn workers
	for i := 0; i < w.concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for {
				// Check context cancellation
				select {
				case <-ctx.Done():
					return
				default:
				}

				// Blocking Pop a server ID from the shared Redis queue
				// Timeout of 2 seconds allows the loop to frequently check context cancellation
				result, err := w.rdb.BLPop(ctx, 2*time.Second, "monitoring:queue").Result()
				if err == redis.Nil {
					// Timeout reached, queue is empty, loop again
					continue
				} else if err != nil {
					if ctx.Err() != nil {
						return
					}
					logger.Log.Sugar().Errorf("[Worker-%d] Failed to pop from queue: %v", workerID, err)
					time.Sleep(1 * time.Second)
					continue
				}
				if len(result) == 2 {
					serverID := result[1]
					w.processServer(ctx, serverID)
				}
			}
		}(i)
	}

	// Wait for all local workers to finish processing their pops
	wg.Wait()
	return nil
}

func (w *workerPool) processServer(ctx context.Context, serverID string) {
	// Get server info (IPv4) from Redis
	redisKey := fmt.Sprintf(infraRedis.ServerInfoKeyFmt, serverID)
	ipv4, err := w.rdb.HGet(ctx, redisKey, "ipv4").Result()
	if err != nil {
		logger.Log.Sugar().Errorf("[Worker] Failed to get IP for server %s: %v\n", serverID, err)
		return
	}
	if ipv4 == "" {
		logger.Log.Sugar().Infof("[Worker] IPv4 is empty for server %s\n", serverID)
		return
	}

	// Perform Ping
	success := w.pinger.Ping(ipv4, w.timeout)

	// Evaluate State Machine
	err = w.monService.Evaluate(ctx, serverID, ipv4, success)
	if err != nil {
		logger.Log.Sugar().Errorf("[Worker] Failed to evaluate state for server %s (IP: %s): %v\n", serverID, ipv4, err)
	}
}
