package handler

import (
	"net/http"
	"strings"

	"sms-identity/internal/infrastructure/security"
)

type ForwardAuthHandler struct {
	authenticator *security.Authenticator
}

func NewForwardAuthHandler(authenticator *security.Authenticator) *ForwardAuthHandler {
	return &ForwardAuthHandler{
		authenticator: authenticator,
	}
}

// Verify is the endpoint called by Traefik ForwardAuth
func (h *ForwardAuthHandler) Verify(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "Unauthorized: No token provided", http.StatusUnauthorized)
		return
	}

	principal, err := h.authenticator.Authenticate(r.Context(), token)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// Set headers for downstream services
	w.Header().Set("X-User-Id", principal.UserID)
	w.Header().Set("X-User-Role", string(principal.RoleCode))
	w.Header().Set("X-User-Email", principal.Email)
	w.Header().Set("X-User-Name", principal.Username)

	// Traefik expects a 200 OK
	w.WriteHeader(http.StatusOK)
}

func extractToken(r *http.Request) string {
	// Try Authorization header first
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Try cookie next
	cookie, err := r.Cookie("jwt_token")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	return ""
}