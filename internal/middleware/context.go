package middleware

import "context"

// UserIDFromContext извлекает userID из контекста.
// Работает и для *http.Request (через r.Context()), и для gRPC (ctx).
func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok && userID != ""
}
