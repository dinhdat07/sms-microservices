package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestMasterKeyAuthMiddleware(t *testing.T) {
	logger := zap.NewNop()
	expectedKey := "secret-key"
	middleware := NewMasterKeyAuthMiddleware(expectedKey, logger)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	handlerToTest := middleware.Middleware(nextHandler)

	t.Run("Valid Key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Master-Key", "secret-key")
		rec := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "OK", rec.Body.String())
	})

	t.Run("Missing Key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "Unauthorized: Invalid or missing X-Master-Key")
	})

	t.Run("Invalid Key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Master-Key", "wrong-key")
		rec := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "Unauthorized: Invalid or missing X-Master-Key")
	})
}
