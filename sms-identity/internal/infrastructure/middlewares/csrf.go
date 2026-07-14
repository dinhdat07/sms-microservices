package middlewares

import (
	"context"
	"strings"

	authv1 "sms-identity/gen/go/auth/v1"
	"sms-identity/internal/infrastructure/security"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// stateChangingMethods are gRPC methods that require CSRF validation.
func stateChangingMethods() map[string]bool {
	return map[string]bool{
		// Auth
		authv1.AuthService_Login_FullMethodName:     true,
		authv1.AuthService_Logout_FullMethodName:    true,
		authv1.AuthService_LogoutAll_FullMethodName: true,
	}
}

func CSRFInterceptor(csrfManager *security.CSRFManager) grpc.UnaryServerInterceptor {
	protectedMethods := stateChangingMethods()

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !protectedMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return handler(ctx, req)
		}

		cookieValues := md.Get("cookie-csrf-token")
		headerValues := md.Get("x-csrf-token")

		cookieToken := ""
		headerToken := ""

		if len(cookieValues) > 0 {
			cookieToken = strings.TrimSpace(cookieValues[0])
		}
		if len(headerValues) > 0 {
			headerToken = strings.TrimSpace(headerValues[0])
		}

		// If neither is present, client may not have cookies yet (e.g., login).
		if cookieToken == "" && headerToken == "" {
			return handler(ctx, req)
		}

		if err := csrfManager.ValidateCSRFToken(cookieToken, headerToken); err != nil {
			return nil, status.Error(codes.PermissionDenied, "CSRF validation failed: "+err.Error())
		}

		return handler(ctx, req)
	}
}