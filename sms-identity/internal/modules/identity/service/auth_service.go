package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"sms-identity/internal/infrastructure/security"
	"sms-identity/internal/modules/identity/domain"
	"sms-identity/internal/modules/identity/repository"
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrUserNotFound        = errors.New("user not found")
	ErrUnauthorized        = errors.New("unauthorized")
)

type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	User         *domain.User
}

type RefreshResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

type AuthService interface {
	Login(ctx context.Context, email, password, ipAddress, userAgent string) (*LoginResult, error)
	Refresh(ctx context.Context, refreshToken string) (*RefreshResult, error)
	Logout(ctx context.Context, sessionIDStr string) error
	LogoutAll(ctx context.Context, userID uint) error
}

type authServiceImpl struct {
	userRepo    repository.UserRepository
	sessionRepo repository.AuthSessionRepository
	refreshRepo repository.RefreshTokenRepository
	revoStore   repository.SessionRevocationStore
	tokenMgr    security.TokenManager
}

func NewAuthService(
	userRepo repository.UserRepository,
	sessionRepo repository.AuthSessionRepository,
	refreshRepo repository.RefreshTokenRepository,
	revoStore repository.SessionRevocationStore,
	tokenMgr security.TokenManager,
) AuthService {
	return &authServiceImpl{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		refreshRepo: refreshRepo,
		revoStore:   revoStore,
		tokenMgr:    tokenMgr,
	}
}

func (s *authServiceImpl) Login(ctx context.Context, email, password, ipAddress, userAgent string) (*LoginResult, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	now := time.Now()
	refreshExpiresAt := now.Add(7 * 24 * time.Hour) // 7 days

	// Generate new raw refresh token
	rawRefreshToken := s.tokenMgr.GenerateRefreshToken()
	hashedToken := s.tokenMgr.HashToken(rawRefreshToken)

	session := &domain.AuthSession{
		UserID:     user.ID,
		ExpiresAt:  refreshExpiresAt,
		LastUsedAt: &now,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	}

	// Transaction logic: create session then refresh token
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	refreshTokenModel := &domain.RefreshToken{
		SessionID: session.ID,
		UserID:    user.ID,
		TokenHash: hashedToken,
		ExpiresAt: refreshExpiresAt,
	}

	if err := s.refreshRepo.Create(ctx, refreshTokenModel); err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	accessToken, err := s.tokenMgr.GenerateAccessToken(user.ID, user.RoleCode, session.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		ExpiresIn:    15 * 60, // 15 mins
		User:         user,
	}, nil
}

func (s *authServiceImpl) Refresh(ctx context.Context, refreshToken string) (*RefreshResult, error) {
	if refreshToken == "" {
		return nil, ErrInvalidRefreshToken
	}

	tokenHash := s.tokenMgr.HashToken(refreshToken)
	foundToken, err := s.refreshRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil || foundToken == nil {
		return nil, ErrInvalidRefreshToken
	}

	if foundToken.RevokedAt != nil {
		// Token reuse detected. Revoke all sessions for security.
		_ = s.LogoutAll(ctx, foundToken.UserID)
		return nil, ErrInvalidRefreshToken
	}

	if foundToken.ExpiresAt.Before(time.Now()) {
		return nil, ErrInvalidRefreshToken
	}

	session, err := s.sessionRepo.FindActiveByID(ctx, foundToken.SessionID)
	if err != nil || session == nil {
		return nil, ErrInvalidRefreshToken
	}

	user, err := s.userRepo.FindByID(ctx, session.UserID)
	if err != nil || user == nil {
		return nil, ErrUserNotFound
	}

	// Generate new tokens
	rawRefreshToken := s.tokenMgr.GenerateRefreshToken()
	newHash := s.tokenMgr.HashToken(rawRefreshToken)

	now := time.Now()
	newRefreshExpiresAt := now.Add(7 * 24 * time.Hour)
	if newRefreshExpiresAt.After(session.ExpiresAt) {
		newRefreshExpiresAt = session.ExpiresAt
	}

	newRefreshTokenModel := &domain.RefreshToken{
		SessionID: session.ID,
		UserID:    user.ID,
		TokenHash: newHash,
		ExpiresAt: newRefreshExpiresAt,
	}

	// In a real app we should use a transaction here, but we simplify for now
	_ = s.refreshRepo.RevokeByID(ctx, foundToken.ID)
	_ = s.refreshRepo.Create(ctx, newRefreshTokenModel)
	_ = s.refreshRepo.MarkReplacement(ctx, foundToken.ID, newRefreshTokenModel.ID)

	accessToken, err := s.tokenMgr.GenerateAccessToken(user.ID, user.RoleCode, session.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new access token: %w", err)
	}

	return &RefreshResult{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		ExpiresIn:    15 * 60,
	}, nil
}

func (s *authServiceImpl) Logout(ctx context.Context, sessionIDStr string) error {
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return ErrInvalidRefreshToken
	}

	session, err := s.sessionRepo.FindActiveByID(ctx, sessionID)
	if err != nil || session == nil {
		return nil // idempotent
	}

	_ = s.sessionRepo.RevokeByID(ctx, sessionID)
	_ = s.refreshRepo.RevokeBySessionID(ctx, sessionID)

	// Mark in Redis
	_ = s.revoStore.MarkRevoked(ctx, session.ID, session.ExpiresAt)

	return nil
}

func (s *authServiceImpl) LogoutAll(ctx context.Context, userID uint) error {
	sessions, err := s.sessionRepo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return err
	}

	_ = s.sessionRepo.RevokeAllByUserID(ctx, userID)
	_ = s.refreshRepo.RevokeByUserID(ctx, userID)

	for _, session := range sessions {
		_ = s.revoStore.MarkRevoked(ctx, session.ID, session.ExpiresAt)
	}

	return nil
}