package grpcserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/Knapptan/Y-URL-shortener/internal/middleware"
	"github.com/Knapptan/Y-URL-shortener/internal/service"
	pb "github.com/Knapptan/Y-URL-shortener/proto"
)

// Server — gRPC-реализация ShortenerService.
// Тонкий фасад: вся бизнес-логика в URLService (том же, что и для HTTP).
type Server struct {
	pb.UnimplementedShortenerServiceServer
	svc     *service.URLService
	baseURL string
}

// NewServer создаёт gRPC-сервер.
func NewServer(svc *service.URLService, baseURL string) *Server {
	return &Server{svc: svc, baseURL: baseURL}
}

// ShortenURL — аналог POST /api/shorten.
func (s *Server) ShortenURL(ctx context.Context, req *pb.URLShortenRequest) (*pb.URLShortenResponse, error) {
	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	id, _, err := s.svc.Shorten(req.GetUrl(), userID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &pb.URLShortenResponse{Result: s.baseURL + "/" + id}, nil
}

// ExpandURL — аналог GET /{id}.
func (s *Server) ExpandURL(_ context.Context, req *pb.URLExpandRequest) (*pb.URLExpandResponse, error) {
	originalURL, ok, deleted, err := s.svc.GetOriginal(req.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !ok || deleted {
		return nil, status.Error(codes.NotFound, "url not found")
	}
	return &pb.URLExpandResponse{Result: originalURL}, nil
}

// ListUserURLs — аналог GET /api/user/urls.
func (s *Server) ListUserURLs(ctx context.Context, _ *emptypb.Empty) (*pb.UserURLsResponse, error) {
	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	records, err := s.svc.GetUserURLs(userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := &pb.UserURLsResponse{}
	for _, rec := range records {
		resp.Url = append(resp.Url, &pb.URLData{
			ShortUrl:    s.baseURL + "/" + rec.ID,
			OriginalUrl: rec.OriginalURL,
		})
	}
	return resp, nil
}
