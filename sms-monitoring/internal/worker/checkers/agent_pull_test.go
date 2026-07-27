package checkers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAgentPullChecker_Check(t *testing.T) {
	checker := NewAgentPullChecker(2 * time.Second)

	t.Run("missing endpoint", func(t *testing.T) {
		result := checker.Check(context.Background(), ServerConfig{})
		assert.False(t, result)
	})

	t.Run("invalid url", func(t *testing.T) {
		result := checker.Check(context.Background(), ServerConfig{"agent_endpoint": "ht\ntp://invalid"})
		assert.False(t, result)
	})

	t.Run("http request fails (network error)", func(t *testing.T) {
		result := checker.Check(context.Background(), ServerConfig{"agent_endpoint": "http://127.0.0.1:0"})
		assert.False(t, result)
	})

	t.Run("http success (200 OK)", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		result := checker.Check(context.Background(), ServerConfig{"agent_endpoint": ts.URL})
		assert.True(t, result)
	})

	t.Run("http failed (500)", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		result := checker.Check(context.Background(), ServerConfig{"agent_endpoint": ts.URL})
		assert.False(t, result)
	})
}
