package grpcserver_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	grpcserver "github.com/Knapptan/Y-URL-shortener/internal/grpcserver"
	"github.com/Knapptan/Y-URL-shortener/internal/middleware"
	"github.com/Knapptan/Y-URL-shortener/internal/model"
	"github.com/Knapptan/Y-URL-shortener/internal/service"
	pb "github.com/Knapptan/Y-URL-shortener/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type miniRepo struct {
	getFunc              func(string) (model.URLRecord, bool, error)
	getByOriginalURLFunc func(string) (string, model.URLRecord, bool, error)
	getByUserIDFunc      func(string) ([]model.URLRecord, error)
	saveFunc             func(string, model.URLRecord) error
}

func (m *miniRepo) Save(id string, rec model.URLRecord) error {
	if m.saveFunc != nil {
		return m.saveFunc(id, rec)
	}
	return nil
}
func (m *miniRepo) Get(id string) (model.URLRecord, bool, error) {
	if m.getFunc != nil {
		return m.getFunc(id)
	}
	return model.URLRecord{}, false, nil
}
func (m *miniRepo) SaveBatch(map[string]model.URLRecord) error { return nil }
func (m *miniRepo) GetByOriginalURL(u string) (string, model.URLRecord, bool, error) {
	if m.getByOriginalURLFunc != nil {
		return m.getByOriginalURLFunc(u)
	}
	return "", model.URLRecord{}, false, nil
}
func (m *miniRepo) GetByUserID(u string) ([]model.URLRecord, error) {
	if m.getByUserIDFunc != nil {
		return m.getByUserIDFunc(u)
	}
	return nil, nil
}
func (m *miniRepo) BatchDeleteByUserID(string, []string) error { return nil }
func (m *miniRepo) CountURLs() (int, error)                    { return 0, nil }
func (m *miniRepo) CountUsers() (int, error)                   { return 0, nil }

func ctxWithUser(id string) context.Context {
	return context.WithValue(context.Background(), middleware.UserIDKey, id)
}

func TestShortenURL(t *testing.T) {
	s := grpcserver.NewServer(service.NewURLService(&miniRepo{}, "http://localhost:8080"), "http://localhost:8080")

	_, err := s.ShortenURL(context.Background(), &pb.URLShortenRequest{Url: "https://ya.ru"})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))

	resp, err := s.ShortenURL(ctxWithUser("u1"), &pb.URLShortenRequest{Url: "https://ya.ru"})
	require.NoError(t, err)
	assert.Contains(t, resp.Result, "http://localhost:8080/")
}

func TestExpandURL(t *testing.T) {
	repo := &miniRepo{
		getFunc: func(_ string) (model.URLRecord, bool, error) {
			return model.URLRecord{OriginalURL: "https://ya.ru"}, true, nil
		},
	}
	s := grpcserver.NewServer(service.NewURLService(repo, "http://localhost:8080"), "http://localhost:8080")

	resp, err := s.ExpandURL(context.Background(), &pb.URLExpandRequest{Id: "abc"})
	require.NoError(t, err)
	assert.Equal(t, "https://ya.ru", resp.Result)
}

func TestExpandURL_NotFound(t *testing.T) {
	s := grpcserver.NewServer(service.NewURLService(&miniRepo{}, "http://localhost:8080"), "http://localhost:8080")
	_, err := s.ExpandURL(context.Background(), &pb.URLExpandRequest{Id: "abc"})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestListUserURLs(t *testing.T) {
	repo := &miniRepo{
		getByUserIDFunc: func(_ string) ([]model.URLRecord, error) {
			return []model.URLRecord{
				{ID: "a", OriginalURL: "https://ya.ru"},
				{ID: "b", OriginalURL: "https://google.com"},
			}, nil
		},
	}
	s := grpcserver.NewServer(service.NewURLService(repo, "http://localhost:8080"), "http://localhost:8080")

	resp, err := s.ListUserURLs(ctxWithUser("u1"), &emptypb.Empty{})
	require.NoError(t, err)
	assert.Len(t, resp.Url, 2)
}
