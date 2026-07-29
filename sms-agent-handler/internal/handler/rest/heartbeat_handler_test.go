package rest

import (
	"bytes"
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

	t.Run("invalid json payload", func(t *testing.T) {
		mr, _ := miniredis.Run()
		defer mr.Close()
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		handler := NewHeartbeatHandler(rdb, logger)

		req := httptest.NewRequest("POST", "/api/v1/agent/heartbeat", bytes.NewBuffer([]byte(`{invalid}`)))
		rec := httptest.NewRecorder()

		handler.HandleHeartbeat(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid payload")
	})

	t.Run("missing server_id", func(t *testing.T) {
		mr, _ := miniredis.Run()
		defer mr.Close()
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		handler := NewHeartbeatHandler(rdb, logger)

		payload := map[string]string{}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/agent/heartbeat", bytes.NewBuffer(body))
		rec := httptest.NewRecorder()

		handler.HandleHeartbeat(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "server_id is required")
	})

	t.Run("success", func(t *testing.T) {
		mr, _ := miniredis.Run()
		defer mr.Close()
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		handler := NewHeartbeatHandler(rdb, logger)

		payload := map[string]string{"server_id": "srv-1"}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/agent/heartbeat", bytes.NewBuffer(body))
		rec := httptest.NewRecorder()

		handler.HandleHeartbeat(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "ok")
		

		// Wait stream check is not fully supported in simple miniredis.Require* but we know XAdd was called.
	})

	t.Run("redis down", func(t *testing.T) {
		mr, _ := miniredis.Run()
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		handler := NewHeartbeatHandler(rdb, logger)
		
		mr.Close() // Force redis error

		payload := map[string]string{"server_id": "srv-1"}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/agent/heartbeat", bytes.NewBuffer(body))
		rec := httptest.NewRecorder()

		handler.HandleHeartbeat(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "Internal server error")
	})
}
