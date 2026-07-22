package middlewares

import (
	"context"

	"sms-management/internal/infrastructure/grpcctx"
	"sms-management/internal/infrastructure/security"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func AuthMetadataInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return handler(ctx, req)
		}

		// Hàm helper lấy giá trị đầu tiên của một khóa
		getVal := func(k string) string {
			if v := md.Get(k); len(v) > 0 {
				return v[0]
			}
			return ""
		}

		principal := security.Principal{
			UserID:   getVal(security.HeaderUserID),
			RoleCode: getVal(security.HeaderUserRole),
			Email:    getVal(security.HeaderUserEmail),
			Username: getVal(security.HeaderUserName),
		}

		if principal.UserID != "" {
			ctx = grpcctx.SetPrincipal(ctx, &principal)
		}

		return handler(ctx, req)
	}
}
