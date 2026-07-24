package middlewares

import (
	"context"
	"errors"
	"strings"

	"sms-identity/internal/infrastructure/security"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func AuthenticationInterceptor(authenticator *security.Authenticator, publicMethods map[string]bool) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if authenticator == nil {
			return nil, status.Error(codes.Internal, "internal server error")
		}

		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		tokenString, err := extractBearerTokenFromMetadata(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}

		principal, err := authenticator.Authenticate(ctx, tokenString)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid authorize format")
		}

		ctx = security.SetPrincipal(ctx, principal)
		return handler(ctx, req)
	}
}

func extractBearerTokenFromMetadata(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("missing token")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", errors.New("missing token")
	}

	authHeader := strings.TrimSpace(values[0])
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" || strings.TrimSpace(parts[1]) == "" {
		return "", errors.New("invalid token")
	}

	return strings.TrimSpace(parts[1]), nil
}
