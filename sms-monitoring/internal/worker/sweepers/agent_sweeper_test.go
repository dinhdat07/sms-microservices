package sweepers

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	infraRedis "sms-monitoring/internal/infrastructure/redis"
	"sms-monitoring/internal/service/mock"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
	"github.com/go-redis/redismock/v9"
)

func TestAgentSweeper_StartAndSweep(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	monService := mocks.NewMonitoringService(t)

	// We set timeout to 120 seconds
	sweeper := NewAgentSweeper(rdb, monService, 10*time.Millisecond, 120)

	// Add data to miniredis
	now := time.Now().Unix()
	
	// Server 1: Expired (Score < now - 120)
	mr.ZAdd(infraRedis.AgentHeartbeatZSetKey, float64(now-130), "srv-1") //nolint:errcheck
	// Server 2: Also Expired (will mock HGet success)
	mr.ZAdd(infraRedis.AgentHeartbeatZSetKey, float64(now-130), "srv-2") //nolint:errcheck
	// Server 3: Not expired
	mr.ZAdd(infraRedis.AgentHeartbeatZSetKey, float64(now-10), "srv-3") //nolint:errcheck

	mr.HSet(fmt.Sprintf(infraRedis.ServerInfoKeyFmt, "srv-2"), infraRedis.ServerInfoFieldIPv4, "1.1.1.1")

	// Expect Evaluate for srv-1 (HGet fails so empty IP)
	monService.On("Evaluate", mock.Anything, "srv-1", "", false).Return(errors.New("eval error")).Once()
	
	// Expect Evaluate for srv-2 (HGet success)
	monService.On("Evaluate", mock.Anything, "srv-2", "1.1.1.1", false).Return(nil).Once()

	ctx, cancel := context.WithCancel(context.Background())
	
	// Start sweeper in goroutine
	go sweeper.Start(ctx)

	// Wait enough time for a tick
	time.Sleep(50 * time.Millisecond)
	cancel() // Stop the sweeper
	time.Sleep(20 * time.Millisecond) // allow shutdown

	// Verify that expired servers were removed from ZSet
	members, _ := mr.ZMembers(infraRedis.AgentHeartbeatZSetKey)
	
	foundSrv1 := false
	foundSrv3 := false
	for _, m := range members {
		if m == "srv-1" {
			foundSrv1 = true
		}
		if m == "srv-3" {
			foundSrv3 = true
		}
	}
	
	if foundSrv1 {
		t.Errorf("Expected srv-1 to be removed from ZSet")
	}
	if !foundSrv3 {
		t.Errorf("Expected srv-3 to remain in ZSet")
	}
	
	monService.AssertExpectations(t)
}

func TestAgentSweeper_ZRangeError(t *testing.T) {
	// Create a mock redis client
	db, mockRedis := redismock.NewClientMock()
	monService := mocks.NewMonitoringService(t)

	sweeper := NewAgentSweeper(db, monService, 10*time.Millisecond, 120)

	// Calculate exact max
	expiredScore := fmt.Sprintf("%d", time.Now().Unix()-120)

	// Simulate error on ZRangeByScore
	mockRedis.ExpectZRangeByScore("server:agent_heartbeats", &redis.ZRangeBy{
		Min: "-inf",
		Max: expiredScore,
	}).SetErr(errors.New("redis error"))

	// Call sweep directly
	sweeper.sweep(context.Background())

	monService.AssertNotCalled(t, "Evaluate")
}
