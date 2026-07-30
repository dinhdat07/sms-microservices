package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestHandleHeartbeat(t *testing.T) {
	logger := zap.NewNop()

	t.Run("missing server_id in context", func(t *testing.T) {
		mr, _ := miniredis.Run()
		defer mr.Close()
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		handler := NewHeartbeatHandler(rdb, logger)

		req := httptest.NewRequest("POST", "/api/v1/agent/heartbeat", bytes.NewBuffer([]byte(`{}`)))
		rec := httptest.NewRecorder()

		handler.HandleHeartbeat(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "server_id is missing from context")
	})



	t.Run("success", func(t *testing.T) {
		mr, _ := miniredis.Run()
		defer mr.Close()
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		handler := NewHeartbeatHandler(rdb, logger)

		payload := map[string]interface{}{"cpu_usage": 50.5}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/agent/heartbeat", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), ServerIDKey, "srv-1")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.HandleHeartbeat(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "ok")
	})

	t.Run("redis down", func(t *testing.T) {
		mr, _ := miniredis.Run()
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		handler := NewHeartbeatHandler(rdb, logger)
		
		mr.Close() // Force redis error

		payload := map[string]interface{}{"cpu_usage": 50.5}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/agent/heartbeat", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), ServerIDKey, "srv-1")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler.HandleHeartbeat(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "Internal server error")
	})
}
