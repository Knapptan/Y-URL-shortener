package service

import (
	"errors"
	"strings"

	"github.com/Knapptan/Y-URL-shortener/internal/repository"
)

// URLService обслуживает запросы на сокращение и получение URL
type URLService struct {
	repo repository.URLRepository
}

// NewURLService конструктор
func NewURLService(repo repository.URLRepository) *URLService {
	return &URLService{repo: repo}
}

// Shorten сохраняет URL и возвращает его короткий идентификатор
func (s *URLService) Shorten(originalURL string) (string, error) {
	// базовая валидация (можно расширить)
	originalURL = strings.TrimSpace(originalURL)
	if originalURL == "" {
		return "", errors.New("empty URL")
	}
	// сохраняем
	id, err := s.repo.Save(originalURL)
	if err != nil {
		return "", err
	}
	return id, nil
}

// GetOriginal возвращает оригинальный URL по ID
func (s *URLService) GetOriginal(id string) (string, bool) {
	if id == "" {
		return "", false
	}
	return s.repo.Get(id)
}
