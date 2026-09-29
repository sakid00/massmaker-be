package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sakid00/massmaker-be/internal/auth"
	"github.com/sakid00/massmaker-be/internal/domain"
)

type contextKey string

const (
	contextUserID contextKey = "userID"
	contextRole   contextKey = "role"
)

func HumanAuth(secret, issuer string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" {
				writeAuthError(w, domain.ErrUnauthorized)
				return
			}
			claims, err := auth.ParseAccess(secret, issuer, token)
			if err != nil {
				writeAuthError(w, domain.ErrUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), contextUserID, claims.Subject)
			ctx = context.WithValue(ctx, contextRole, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func OptionalHumanAuth(secret, issuer string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}
			claims, err := auth.ParseAccess(secret, issuer, token)
			if err != nil {
				writeAuthError(w, domain.ErrUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), contextUserID, claims.Subject)
			ctx = context.WithValue(ctx, contextRole, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func StaffAuth(parse func(string) error) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" || parse(token) != nil {
				writeAuthError(w, domain.ErrUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func UserFromCtx(ctx context.Context) (id, role string) {
	id, _ = ctx.Value(contextUserID).(string)
	role, _ = ctx.Value(contextRole).(string)
	return id, role
}

func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}

func writeAuthError(w http.ResponseWriter, err *domain.AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": err})
}
