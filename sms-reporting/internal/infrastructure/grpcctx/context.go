package grpcctx

import (
	"context"
	"errors"

	"sms-reporting/internal/infrastructure/security"
)

type contextKey string

const principalContextKey contextKey = "principal"

func SetPrincipal(ctx context.Context, principal *security.Principal) context.Context {
	return context.WithValue(ctx, principalContextKey, principal)
}

func GetPrincipal(ctx context.Context) (*security.Principal, bool) {
	v := ctx.Value(principalContextKey)
	if v == nil {
		return nil, false
	}

	principal, ok := v.(*security.Principal)
	return principal, ok
}

func GetActorFromCtx(ctx context.Context) (*security.Principal, error) {
	actor, ok := GetPrincipal(ctx)
	if !ok {
		return nil, errors.New("unauthorized")
	}
	return actor, nil
}
