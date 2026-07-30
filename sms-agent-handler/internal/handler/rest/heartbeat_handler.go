package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type HeartbeatHandler struct {
	rdb    redis.UniversalClient
	logger *zap.Logger
}

func NewHeartbeatHandler(rdb redis.UniversalClient, logger *zap.Logger) *HeartbeatHandler {
	return &HeartbeatHandler{
		rdb:    rdb,
		logger: logger,
	}
}

func (h *HeartbeatHandler) HandleHeartbeat(w http.ResponseWriter, r *http.Request) {
	serverID, ok := r.Context().Value(ServerIDKey).(string)
	if !ok || serverID == "" {
		http.Error(w, "server_id is missing from context", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	now := time.Now().Unix()

	// Update timestamp in ZSET
	zsetKey := "monitoring:agent:heartbeats"
	err := h.rdb.ZAdd(ctx, zsetKey, redis.Z{
		Score:  float64(now),
		Member: serverID,
	}).Err()
	if err != nil {
		h.logger.Error("Failed to update heartbeat in Redis", zap.Error(err), zap.String("server_id", serverID))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Publish event to Redis Stream
	streamKey := "sms.events.heartbeat"
	err = h.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey,
		Values: map[string]interface{}{
			"server_id": serverID,
		},
	}).Err()

	if err != nil {
		h.logger.Error("Failed to publish heartbeat event", zap.Error(err), zap.String("server_id", serverID))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status": "ok"}`))
}
