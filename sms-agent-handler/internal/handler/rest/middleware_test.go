package rest

import (

	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestAgentJWTAuthMiddleware(t *testing.T) {
	logger := zap.NewNop()
	secret := "my-secret"
	middleware := NewAgentJWTAuthMiddleware(secret, logger)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverID := r.Context().Value(ServerIDKey)
		assert.Equal(t, "srv-1", serverID)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	handlerToTest := middleware.Middleware(nextHandler)

	generateToken := func(serverID, boundIP string, addExpiry bool) string {
		claims := jwt.MapClaims{
			"server_id": serverID,
			"bound_ip":  boundIP,
		}
		if addExpiry {
			claims["exp"] = time.Now().Add(time.Hour).Unix()
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		str, _ := token.SignedString([]byte(secret))
		return str
	}

	t.Run("Valid Token and IP", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+generateToken("srv-1", "1.2.3.4", false))
		req.Header.Set("X-Forwarded-For", "1.2.3.4")
		rec := httptest.NewRecorder()
		handlerToTest.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("Valid Token IP Mismatch", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+generateToken("srv-1", "1.2.3.4", false))
		req.Header.Set("X-Forwarded-For", "4.5.6.7")
		rec := httptest.NewRecorder()
		handlerToTest.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("Missing Authorization Header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		handlerToTest.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Invalid Token Signature", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		wrongSecretToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"server_id": "srv-1", "bound_ip": "1.2.3.4"})
		str, _ := wrongSecretToken.SignedString([]byte("wrong-secret"))
		req.Header.Set("Authorization", "Bearer "+str)
		rec := httptest.NewRecorder()
		handlerToTest.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
	
	t.Run("Multiple IPs in X-Forwarded-For", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+generateToken("srv-1", "1.2.3.4", false))
		req.Header.Set("X-Forwarded-For", "5.6.7.8, 1.2.3.4")
		rec := httptest.NewRecorder()
		handlerToTest.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code) // Should pick 1.2.3.4
	})
}
