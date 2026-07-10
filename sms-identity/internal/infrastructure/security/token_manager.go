package security

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"sms-identity/internal/modules/identity/domain"
)

type TokenManager interface {
	HashToken(token string) string
	GenerateAccessToken(userID uint, roleCode domain.RoleCode, sessionID uuid.UUID) (string, error)
	GenerateRefreshToken() string
}

type tokenManagerImpl struct {
	jwtSecret string
}

func NewTokenManager(jwtSecret string) TokenManager {
	return &tokenManagerImpl{jwtSecret: jwtSecret}
}

func (t *tokenManagerImpl) HashToken(token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}

func (t *tokenManagerImpl) GenerateAccessToken(userID uint, roleCode domain.RoleCode, sessionID uuid.UUID) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":    fmt.Sprintf("%d", userID),
		"role_code":  roleCode,
		"session_id": sessionID.String(),
		"exp":        time.Now().Add(15 * time.Minute).Unix(),
		"iat":        time.Now().Unix(),
	})
	return token.SignedString([]byte(t.jwtSecret))
}

func (t *tokenManagerImpl) GenerateRefreshToken() string {
	return uuid.New().String()
}