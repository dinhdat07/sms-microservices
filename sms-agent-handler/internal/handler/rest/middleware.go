package rest

import (
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type MasterKeyAuthMiddleware struct {
	expectedKey string
	logger      *zap.Logger
}

func NewMasterKeyAuthMiddleware(expectedKey string, logger *zap.Logger) *MasterKeyAuthMiddleware {
	return &MasterKeyAuthMiddleware{
		expectedKey: strings.TrimSpace(expectedKey),
		logger:      logger,
	}
}

func (m *MasterKeyAuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providedKey := r.Header.Get("X-Master-Key")
		
		if providedKey == "" || providedKey != m.expectedKey {
			m.logger.Warn("Unauthorized heartbeat attempt", 
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("provided_key", providedKey))
			http.Error(w, "Unauthorized: Invalid or missing X-Master-Key", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
