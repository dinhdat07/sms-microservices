package rest

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type contextKey string
const ServerIDKey contextKey = "server_id"

type AgentJWTAuthMiddleware struct {
	secret []byte
	logger *zap.Logger
}

func NewAgentJWTAuthMiddleware(secret string, logger *zap.Logger) *AgentJWTAuthMiddleware {
	return &AgentJWTAuthMiddleware{
		secret: []byte(strings.TrimSpace(secret)),
		logger: logger,
	}
}

func (m *AgentJWTAuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			m.logger.Warn("Unauthorized heartbeat attempt: missing Bearer token", zap.String("remote_addr", r.RemoteAddr))
			http.Error(w, "Unauthorized: Missing Bearer token", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return m.secret, nil
		})

		if err != nil || !token.Valid {
			m.logger.Warn("Unauthorized heartbeat attempt: invalid token", zap.String("remote_addr", r.RemoteAddr), zap.Error(err))
			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Unauthorized: Invalid claims", http.StatusUnauthorized)
			return
		}

		serverID, ok := claims["server_id"].(string)
		if !ok || serverID == "" {
			http.Error(w, "Unauthorized: missing server_id claim", http.StatusUnauthorized)
			return
		}

		boundIP, ok := claims["bound_ip"].(string)
		if !ok || boundIP == "" {
			m.logger.Warn("Unauthorized: missing bound_ip claim", zap.String("server_id", serverID))
			http.Error(w, "Unauthorized: missing bound_ip claim", http.StatusUnauthorized)
			return
		}

		clientIP := getClientIP(r)
		if boundIP != clientIP {
			m.logger.Warn("IP Binding validation failed", 
				zap.String("server_id", serverID),
				zap.String("bound_ip", boundIP),
				zap.String("client_ip", clientIP))
			http.Error(w, "Forbidden: IP mismatch", http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), ServerIDKey, serverID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		// Traefik appends the real client IP to the end of the X-Forwarded-For chain.
		// Taking the last element prevents HTTP Header spoofing.
		return strings.TrimSpace(ips[len(ips)-1])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	// Fallback to RemoteAddr (usually IP:port)
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		addr = addr[:idx] // strip port
	}
	// Remove brackets if IPv6
	addr = strings.Trim(addr, "[]")
	return addr
}
