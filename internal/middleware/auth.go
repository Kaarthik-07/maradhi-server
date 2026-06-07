package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/yourusername/maradhi-api/internal/auth"
)

type contextKey string

const UserIDKey contextKey = "user_id"

func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				jsonErr(w, http.StatusUnauthorized, "missing authorization header")
				return
			}
			claims, err := auth.VerifyToken(jwtSecret, strings.TrimPrefix(h, "Bearer "))
			if err != nil {
				jsonErr(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, claims.Sub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(UserIDKey).(string)
	return v
}

func jsonErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"message":"` + msg + `"}`))
}
