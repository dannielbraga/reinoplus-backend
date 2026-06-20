package middleware

import (
	"context"
	"net/http"
	"strings"

	jwtmanager "github.com/reinoplus/reinoplus/internal/auth/jwt"
	"github.com/reinoplus/reinoplus/internal/domain"
	"github.com/reinoplus/reinoplus/internal/httputil"
	"go.uber.org/zap"
)

type contextKey string

const UserIDKey contextKey = "userID"
const UserRoleKey contextKey = "userRole"

func Auth(jwtManager *jwtmanager.Manager, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" || !strings.HasPrefix(header, "Bearer ") {
				httputil.WriteError(w, logger, domain.ErrUnauthorized)
				return
			}

			token := strings.TrimPrefix(header, "Bearer ")
			claims, err := jwtManager.ParseAccessToken(token)
			if err != nil {
				httputil.WriteError(w, logger, domain.ErrUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UserRoleKey, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireAdmin(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if GetUserRole(r.Context()) != string(domain.UserRoleAdmin) {
				httputil.WriteError(w, logger, domain.ErrForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func GetUserID(ctx context.Context) string {
	value, _ := ctx.Value(UserIDKey).(string)
	return value
}

func GetUserRole(ctx context.Context) string {
	if role, ok := ctx.Value(UserRoleKey).(domain.UserRole); ok {
		return string(role)
	}
	value, _ := ctx.Value(UserRoleKey).(string)
	return value
}
