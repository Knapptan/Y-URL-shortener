package middleware

import (
	"context"
	"net/http"

	"github.com/Knapptan/Y-URL-shortener/internal/auth"
)

type contextKey string

const UserIDKey contextKey = "userID"

// AuthMiddleware проверяет или создаёт куку с userID.
func AuthMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(auth.CookieName)
			if err == http.ErrNoCookie {
				// Куки нет – создаём нового пользователя
				userID := auth.GenerateUserID()
				auth.SetUserCookie(w, userID, secret)
				ctx := context.WithValue(r.Context(), UserIDKey, userID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Кука есть – проверяем подпись
			userID, err := auth.VerifyUserID(cookie.Value, secret)
			if err != nil {
				// Подпись невалидна – 401
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			if userID == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
