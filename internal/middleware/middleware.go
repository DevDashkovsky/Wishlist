package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	myjwt "wishlist-api/internal/jwt"
)

type contextKey string

const userIDKey contextKey = "user_id"

func Auth(jwtManager *myjwt.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				unauthorized(w)
				return
			}

			token := strings.TrimPrefix(header, "Bearer ")
			if token == header {
				unauthorized(w)
				return
			}

			userID, err := jwtManager.Parse(token)
			if err != nil {
				unauthorized(w)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserID(ctx context.Context) int64 {
	id, _ := ctx.Value(userIDKey).(int64)
	return id
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}
