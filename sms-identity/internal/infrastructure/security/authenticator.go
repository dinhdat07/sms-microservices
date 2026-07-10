package security

import (
	"context"
	"errors"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

type Authenticator struct {
	secret      []byte
	redisClient redis.UniversalClient
}

func NewAuthenticator(secret string, redisClient redis.UniversalClient) *Authenticator {
	return &Authenticator{
		secret:      []byte(secret),
		redisClient: redisClient,
	}
}

func (a *Authenticator) Authenticate(ctx context.Context, tokenString string) (*Principal, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&claims{},
		func(t *jwt.Token) (interface{}, error) {
			return a.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	parsedClaims, ok := token.Claims.(*claims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	if a.redisClient != nil && parsedClaims.SessionID != "" {
		revokedKey := "revoked_session:" + parsedClaims.SessionID
		isRevoked, err := a.redisClient.Exists(ctx, revokedKey).Result()
		if err == nil && isRevoked > 0 {
			return nil, errors.New("session is already revoked")
		}
	}

	principal := &Principal{
		UserID:    parsedClaims.UserID,
		Username:  parsedClaims.Username,
		Email:     parsedClaims.Email,
		RoleID:    parsedClaims.RoleID,
		RoleCode:  parsedClaims.RoleCode,
		SessionID: parsedClaims.SessionID,
	}

	return principal, nil
}