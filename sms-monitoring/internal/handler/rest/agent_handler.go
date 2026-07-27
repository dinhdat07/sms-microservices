package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"sms-monitoring/internal/infrastructure/logger"
	infraRedis "sms-monitoring/internal/infrastructure/redis"
	"sms-monitoring/internal/service"

	"github.com/redis/go-redis/v9"
)

type AgentHandler struct {
	rdb         redis.UniversalClient
	monService  service.MonitoringService
	heartbeatCh chan string
}

func NewAgentHandler(rdb redis.UniversalClient, monService service.MonitoringService) *AgentHandler {
	return &AgentHandler{
		rdb:         rdb,
		monService:  monService,
		heartbeatCh: make(chan string, 1000), // Buffer to handle spikes
	}
}

func (h *AgentHandler) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case serverID := <-h.heartbeatCh:
				// Fetch IP to pass to Evaluate
				redisKey := fmt.Sprintf(infraRedis.ServerInfoKeyFmt, serverID)
				ipv4, err := h.rdb.HGet(ctx, redisKey, "ipv4").Result()
				if err != nil {
					ipv4 = "" // fallback
				}

				// Trigger UP state evaluation
				if err := h.monService.Evaluate(ctx, serverID, ipv4, true); err != nil {
					logger.Log.Sugar().Errorf("[AgentHandler] Failed to evaluate state for server %s: %v", serverID, err)
				}
			}
		}
	}()
}

func (h *AgentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ServerID string `json:"server_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if req.ServerID == "" {
		http.Error(w, "server_id is required", http.StatusBadRequest)
		return
	}

	// Save current timestamp to Redis ZSET
	err := h.rdb.ZAdd(r.Context(), infraRedis.AgentHeartbeatZSetKey, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: req.ServerID,
	}).Err()
	if err != nil {
		logger.Log.Sugar().Errorf("Failed to update heartbeat for %s: %v", req.ServerID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Trigger async state evaluation
	select {
	case h.heartbeatCh <- req.ServerID:
	default:
		logger.Log.Sugar().Warnf("Agent heartbeat channel full, skipping active evaluation for %s", req.ServerID)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
