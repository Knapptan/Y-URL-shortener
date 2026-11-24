// Package grpcserver реализует gRPC-сервер сервиса сокращения URL.
package grpcserver

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/Knapptan/Y-URL-shortener/internal/auth"
	"github.com/Knapptan/Y-URL-shortener/internal/middleware"
)

// AuthInterceptor проверяет metadata.authorization, верифицирует подписанную
// куку user_id и кладёт userID в контекст.
//
// Поддерживаемые форматы значения authorization:
//   - "<signed>"
//   - "user_id=<signed>"
//   - "Bearer <signed>"
func AuthInterceptor(secret string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}
		values := md.Get("authorization")
		if len(values) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization")
		}

		raw := strings.TrimSpace(values[0])
		raw = strings.TrimPrefix(raw, "Bearer ")
		raw = strings.TrimPrefix(raw, "user_id=")

		userID, err := auth.VerifyUserID(raw, secret)
		if err != nil || userID == "" {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization")
		}

		ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
		return handler(ctx, req)
	}
}
