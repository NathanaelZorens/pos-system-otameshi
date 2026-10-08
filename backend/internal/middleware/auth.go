package middleware

import (
	"net/http"

	"github.com/pos-system-otameshi/backend/internal/auth"
	"github.com/pos-system-otameshi/backend/internal/httpjson"
)

func Auth(tokens *auth.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := auth.BearerToken(r)
			if token == "" {
				httpjson.Error(w, http.StatusUnauthorized, "missing token")
				return
			}
			claims, err := tokens.Parse(token)
			if err != nil {
				httpjson.Error(w, http.StatusUnauthorized, "invalid token")
				return
			}
			next.ServeHTTP(w, auth.WithClaims(r, claims))
		})
	}
}

func RequireRole(roles ...auth.Role) func(http.Handler) http.Handler {
	set := make(map[auth.Role]struct{}, len(roles))
	for _, r := range roles {
		set[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.ClaimsFromRequest(r)
			if !ok {
				httpjson.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			if _, ok := set[claims.Role]; !ok {
				httpjson.Error(w, http.StatusForbidden, "forbidden")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
