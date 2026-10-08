package auth

import "context"

func contextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

func claimsFromContext(ctx context.Context) (*Claims, bool) {
	v := ctx.Value(claimsKey)
	if v == nil {
		return nil, false
	}
	c, ok := v.(*Claims)
	return c, ok
}
