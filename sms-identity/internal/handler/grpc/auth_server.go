package grpc

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	authv1 "sms-identity/gen/go/auth/v1"
	"sms-identity/internal/infrastructure/security"

	"sms-identity/internal/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func authCookie(name, value string, maxAge int, httpOnly bool) string {
	return (&http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: httpOnly,
		SameSite: http.SameSiteLaxMode,
	}).String()
}

func clearCookie(name string) string {
	return (&http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: name != "csrf_token",
		SameSite: http.SameSiteLaxMode,
	}).String()
}

type AuthServer struct {
	authv1.UnimplementedAuthServiceServer
	authService service.AuthService
}

func NewAuthServer(authService service.AuthService) *AuthServer {
	return &AuthServer{authService: authService}
}

func getClientIP(ctx context.Context) string {
	if p, ok := peer.FromContext(ctx); ok {
		return p.Addr.String()
	}
	return ""
}

func getUserAgent(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if agents := md.Get("grpcgateway-user-agent"); len(agents) > 0 {
			return agents[0]
		}
		if agents := md.Get("user-agent"); len(agents) > 0 {
			return agents[0]
		}
	}
	return ""
}

func (s *AuthServer) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	if req == nil || req.Identifier == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "identifier and password are required")
	}

	ipAddress := getClientIP(ctx)
	userAgent := getUserAgent(ctx)

	result, err := s.authService.Login(ctx, req.Identifier, req.Password, ipAddress, userAgent)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}
		return nil, status.Error(codes.Internal, "failed to login")
	}

	csrfManager := security.NewCSRFManager()
	csrfToken, err := csrfManager.GenerateCSRFToken()
	if err == nil {
		_ = grpc.SetHeader(ctx, metadata.Pairs(
			"Set-Cookie-Access-Token", authCookie("access_token", result.AccessToken, int(result.ExpiresIn), true),
			"Set-Cookie-Refresh-Token", authCookie("refresh_token", result.RefreshToken, 604800, true),
			"Set-Cookie-Csrf-Token", authCookie("csrf_token", csrfToken, 86400, false),
		))
	}

	return &authv1.LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    result.ExpiresIn,
	}, nil
}

func (s *AuthServer) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	principal, ok := security.GetPrincipal(ctx)
	if ok && principal != nil && principal.SessionID != "" {
		_ = s.authService.Logout(ctx, principal.SessionID)
	}

	_ = grpc.SetHeader(ctx, metadata.Pairs(
		"Clear-Cookie", clearCookie("access_token"),
		"Clear-Cookie", clearCookie("refresh_token"),
		"Clear-Cookie", clearCookie("csrf_token"),
	))

	return &authv1.LogoutResponse{
		Message: "Logout successful",
	}, nil
}

func (s *AuthServer) RefreshToken(ctx context.Context, req *authv1.RefreshRequest) (*authv1.RefreshResponse, error) {
	if req == nil || req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required")
	}

	result, err := s.authService.Refresh(ctx, req.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRefreshToken) {
			return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
		}
		return nil, status.Error(codes.Internal, "failed to refresh token")
	}

	csrfManager := security.NewCSRFManager()
	csrfToken, err := csrfManager.GenerateCSRFToken()
	if err == nil {
		_ = grpc.SetHeader(ctx, metadata.Pairs(
			"Set-Cookie-Access-Token", authCookie("access_token", result.AccessToken, int(result.ExpiresIn), true),
			"Set-Cookie-Refresh-Token", authCookie("refresh_token", result.RefreshToken, 604800, true),
			"Set-Cookie-Csrf-Token", authCookie("csrf_token", csrfToken, 86400, false),
		))
	}

	return &authv1.RefreshResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    result.ExpiresIn,
	}, nil
}

func (s *AuthServer) LogoutAll(ctx context.Context, req *authv1.LogoutAllRequest) (*authv1.LogoutAllResponse, error) {
	principal, ok := security.GetPrincipal(ctx)
	if ok && principal != nil && principal.UserID != "" {
		var userID uint
		if _, err := fmt.Sscanf(principal.UserID, "%d", &userID); err != nil {
			return nil, fmt.Errorf("invalid user id format in token: %w", err)
		}
		if userID > 0 {
			_ = s.authService.LogoutAll(ctx, userID)
		}
	}

	_ = grpc.SetHeader(ctx, metadata.Pairs(
		"Clear-Cookie", clearCookie("access_token"),
		"Clear-Cookie", clearCookie("refresh_token"),
		"Clear-Cookie", clearCookie("csrf_token"),
	))

	return &authv1.LogoutAllResponse{
		Message: "Logout all successful",
	}, nil
}
