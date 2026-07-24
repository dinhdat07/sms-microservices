package security

import (
	"context"
	"errors"
)

type contextKey string

const principalContextKey contextKey = "principal"

func SetPrincipal(ctx context.Context, principal *Principal) context.Context {
	return context.WithValue(ctx, principalContextKey, principal)
}

func GetPrincipal(ctx context.Context) (*Principal, bool) {
	v := ctx.Value(principalContextKey)
	if v == nil {
		return nil, false
	}

	principal, ok := v.(*Principal)
	return principal, ok
}

func GetActorFromCtx(ctx context.Context) (*Principal, error) {
	actor, ok := GetPrincipal(ctx)
	if !ok {
		return nil, errors.New("unauthorized")
	}
	return actor, nil
}
