package grpcserver

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authv1 "sms-identity/gen/go/auth/v1"
	"sms-identity/internal/infrastructure/security"
	"sms-identity/internal/service"
	"sms-identity/internal/infrastructure/grpcctx"
)

type mockAuthSvc struct {
	mock.Mock
}

func (m *mockAuthSvc) Login(ctx context.Context, identifier, password, ipAddress, userAgent string) (*service.LoginResult, error) {
	args := m.Called(ctx, identifier, password, ipAddress, userAgent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.LoginResult), args.Error(1)
}

func (m *mockAuthSvc) Logout(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *mockAuthSvc) Refresh(ctx context.Context, refreshToken string) (*service.RefreshResult, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.RefreshResult), args.Error(1)
}

func (m *mockAuthSvc) LogoutAll(ctx context.Context, userID uint) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func TestAuthServer_Login(t *testing.T) {
	svc := new(mockAuthSvc)
	srv := NewAuthServer(svc)

	t.Run("nil request", func(t *testing.T) {
		_, err := srv.Login(context.Background(), nil)
		assert.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("success", func(t *testing.T) {
		svc.On("Login", mock.Anything, "user", "pass", "", "").
			Return(&service.LoginResult{AccessToken: "acc", RefreshToken: "ref", ExpiresIn: 3600}, nil).Once()

		resp, err := srv.Login(context.Background(), &authv1.LoginRequest{Identifier: "user", Password: "pass"})
		assert.NoError(t, err)
		assert.Equal(t, "acc", resp.AccessToken)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		svc.On("Login", mock.Anything, "user", "wrong", "", "").
			Return(nil, service.ErrInvalidCredentials).Once()

		_, err := srv.Login(context.Background(), &authv1.LoginRequest{Identifier: "user", Password: "wrong"})
		assert.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("internal error", func(t *testing.T) {
		svc.On("Login", mock.Anything, "user", "fail", "", "").
			Return(nil, errors.New("db error")).Once()

		_, err := srv.Login(context.Background(), &authv1.LoginRequest{Identifier: "user", Password: "fail"})
		assert.Error(t, err)
		assert.Equal(t, codes.Internal, status.Code(err))
	})
}

func TestAuthServer_Logout(t *testing.T) {
	svc := new(mockAuthSvc)
	srv := NewAuthServer(svc)

	t.Run("success with principal", func(t *testing.T) {
		ctx := grpcctx.SetPrincipal(context.Background(), &security.Principal{SessionID: "sess-1"})
		svc.On("Logout", mock.Anything, "sess-1").Return(nil).Once()

		resp, err := srv.Logout(ctx, &authv1.LogoutRequest{})
		assert.NoError(t, err)
		assert.Equal(t, "Logout successful", resp.Message)
	})

	t.Run("success without principal", func(t *testing.T) {
		resp, err := srv.Logout(context.Background(), &authv1.LogoutRequest{})
		assert.NoError(t, err)
		assert.Equal(t, "Logout successful", resp.Message)
	})
}

func TestAuthServer_RefreshToken(t *testing.T) {
	svc := new(mockAuthSvc)
	srv := NewAuthServer(svc)

	t.Run("nil request", func(t *testing.T) {
		_, err := srv.RefreshToken(context.Background(), nil)
		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		svc.On("Refresh", mock.Anything, "token").
			Return(&service.RefreshResult{AccessToken: "new-acc", RefreshToken: "new-ref", ExpiresIn: 3600}, nil).Once()

		resp, err := srv.RefreshToken(context.Background(), &authv1.RefreshRequest{RefreshToken: "token"})
		assert.NoError(t, err)
		assert.Equal(t, "new-acc", resp.AccessToken)
	})

	t.Run("invalid token", func(t *testing.T) {
		svc.On("Refresh", mock.Anything, "bad").
			Return(nil, service.ErrInvalidRefreshToken).Once()

		_, err := srv.RefreshToken(context.Background(), &authv1.RefreshRequest{RefreshToken: "bad"})
		assert.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})
}

func TestAuthServer_LogoutAll(t *testing.T) {
	svc := new(mockAuthSvc)
	srv := NewAuthServer(svc)

	t.Run("success with valid principal", func(t *testing.T) {
		ctx := grpcctx.SetPrincipal(context.Background(), &security.Principal{UserID: "1"})
		svc.On("LogoutAll", mock.Anything, uint(1)).Return(nil).Once()

		resp, err := srv.LogoutAll(ctx, &authv1.LogoutAllRequest{})
		assert.NoError(t, err)
		assert.Equal(t, "Logout all successful", resp.Message)
	})

	t.Run("invalid user ID", func(t *testing.T) {
		ctx := grpcctx.SetPrincipal(context.Background(), &security.Principal{UserID: "abc"})
		
		_, err := srv.LogoutAll(ctx, &authv1.LogoutAllRequest{})
		assert.Error(t, err)
	})
}