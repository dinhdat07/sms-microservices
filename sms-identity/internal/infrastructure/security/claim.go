package security

import (
	"github.com/golang-jwt/jwt/v5"
	"sms-identity/internal/modules/identity/domain"
)

type claims struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	RoleID    string          `json:"role_id"`
	RoleCode  domain.RoleCode `json:"role_code"`
	jwt.RegisteredClaims
}