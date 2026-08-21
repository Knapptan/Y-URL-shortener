package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/Knapptan/Y-URL-shortener/internal/model"
	"github.com/Knapptan/Y-URL-shortener/internal/repository"
)

// URLService предоставляет методы для сокращения и получения URL.
type URLService struct {
	repo    repository.URLRepository
	baseURL string
}

// BatchItem представляет один элемент для пакетного сокращения.
type BatchItem struct {
	CorrelationID string
	OriginalURL   string
}

// BatchResult представляет результат обработки одного элемента пакетного запроса.
type BatchResult struct {
	CorrelationID string
	ShortURL      string
}

// NewURLService создаёт новый экземпляр URLService.
func NewURLService(repo repository.URLRepository, baseURL string) *URLService {
	return &URLService{repo: repo, baseURL: baseURL}
}

// generateShortID генерирует случайный короткий ID длиной 8 символов.
func (s *URLService) generateShortID() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:8], nil
}

// Shorten обрабатывает запрос на сокращение одного URL.
func (s *URLService) Shorten(originalURL string) (string, bool, error) {
	originalURL = strings.TrimSpace(originalURL)
	if originalURL == "" {
		return "", false, errors.New("empty URL")
	}

	// Проверяем существование
	if existingID, _, ok := s.repo.GetByOriginalURL(originalURL); ok {
		return existingID, true, nil
	}

	// Генерируем новый ID
	id, err := s.generateShortID()
	if err != nil {
		return "", false, err
	}
	record := model.URLRecord{OriginalURL: originalURL}
	if err := s.repo.Save(id, record); err != nil {
		return "", false, err
	}
	return id, false, nil
}

// GetOriginal возвращает оригинальный URL по его короткому ID.
func (s *URLService) GetOriginal(id string) (string, bool) {
	if id == "" {
		return "", false
	}
	record, ok := s.repo.Get(id)
	if !ok {
		return "", false
	}
	return record.OriginalURL, true
}

// ShortenBatch обрабатывает пакетный запрос на сокращение нескольких URL.
func (s *URLService) ShortenBatch(items []BatchItem) ([]BatchResult, error) {
	if len(items) == 0 {
		return nil, errors.New("empty batch")
	}
	batch := make(map[string]model.URLRecord)
	results := make([]BatchResult, 0, len(items))
	for _, item := range items {
		id, err := s.generateShortID()
		if err != nil {
			return nil, err
		}

		batch[id] = model.URLRecord{OriginalURL: item.OriginalURL}
		results = append(results, BatchResult{
			CorrelationID: item.CorrelationID,
			ShortURL:      s.baseURL + "/" + id,
		})
	}
	if err := s.repo.SaveBatch(batch); err != nil {
		return nil, err
	}
	return results, nil
}
