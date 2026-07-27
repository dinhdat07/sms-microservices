package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"sms-monitoring/internal/infrastructure/logger"
	infraRedis "sms-monitoring/internal/infrastructure/redis"

	"github.com/redis/go-redis/v9"
)

type AgentHandler struct {
	rdb redis.UniversalClient
}

func NewAgentHandler(rdb redis.UniversalClient) *AgentHandler {
	return &AgentHandler{rdb: rdb}
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

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
