package handler

import (
	"net/http"
	"strings"

	"sms-identity/internal/infrastructure/security"
)

type ForwardAuthHandler struct {
	authenticator *security.Authenticator
	csrfManager   *security.CSRFManager
}

func NewForwardAuthHandler(authenticator *security.Authenticator, csrfManager *security.CSRFManager) *ForwardAuthHandler {
	return &ForwardAuthHandler{
		authenticator: authenticator,
		csrfManager:   csrfManager,
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

	// CSRF Check for state-changing methods - ONLY if authenticated via Cookie
	if r.Header.Get("Authorization") == "" {
		if err := h.validateCSRF(r); err != nil {
			http.Error(w, "Forbidden: CSRF validation failed - "+err.Error(), http.StatusForbidden)
			return
		}
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

func (h *ForwardAuthHandler) validateCSRF(r *http.Request) error {
	method := r.Header.Get("X-Forwarded-Method")
	method = strings.ToUpper(method)
	
	// Only state-changing methods require CSRF validation
	if method != "POST" && method != "PUT" && method != "DELETE" && method != "PATCH" {
		return nil
	}

	cookieValues := r.Header.Values("Cookie")
	headerToken := r.Header.Get("X-Csrf-Token")

	cookieToken := ""
	for _, cookieStr := range cookieValues {
		if strings.Contains(cookieStr, "csrf_token=") {
			parts := strings.Split(cookieStr, ";")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if strings.HasPrefix(part, "csrf_token=") {
					cookieToken = strings.TrimPrefix(part, "csrf_token=")
				}
			}
		}
	}

	return h.csrfManager.ValidateCSRFToken(cookieToken, headerToken)
}