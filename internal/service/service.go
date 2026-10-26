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
	existingID, _, ok, err := s.repo.GetByOriginalURL(originalURL)
	if err != nil {
		return "", false, err
	}
	if ok {
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
func (s *URLService) GetOriginal(id string) (string, bool, error) {
	if id == "" {
		return "", false, nil // пустой ID – это не ошибка, просто не найдено
	}
	record, ok, err := s.repo.Get(id)
	if err != nil {
		return "", false, err // ошибка репозитория (БД)
	}
	if !ok {
		return "", false, nil // запись не найдена, ошибки нет
	}
	return record.OriginalURL, true, nil
}

// ShortenBatch обрабатывает пакетный запрос на сокращение нескольких URL.
func (s *URLService) ShortenBatch(items []BatchItem) ([]BatchResult, error) {
	if len(items) == 0 {
		return nil, errors.New("empty batch")
	}

	results := make([]BatchResult, 0, len(items))
	batch := make(map[string]model.URLRecord)
	processed := make(map[string]string)

	for _, item := range items {
		// Проверяем, не обрабатывали ли уже этот URL в этом батче
		if existingID, ok := processed[item.OriginalURL]; ok {
			results = append(results, BatchResult{
				CorrelationID: item.CorrelationID,
				ShortURL:      s.baseURL + "/" + existingID,
			})
			continue
		}

		// Проверяем существование в хранилище
		existingID, _, found, err := s.repo.GetByOriginalURL(item.OriginalURL)
		if err != nil {
			return nil, err
		}
		if found {
			processed[item.OriginalURL] = existingID
			results = append(results, BatchResult{
				CorrelationID: item.CorrelationID,
				ShortURL:      s.baseURL + "/" + existingID,
			})
			continue
		}

		// Новый URL: генерируем ID
		id, err := s.generateShortID()
		if err != nil {
			return nil, err
		}
		batch[id] = model.URLRecord{OriginalURL: item.OriginalURL}
		processed[item.OriginalURL] = id
		results = append(results, BatchResult{
			CorrelationID: item.CorrelationID,
			ShortURL:      s.baseURL + "/" + id,
		})
	}

	// Если есть новые записи – сохраняем их атомарно
	if len(batch) > 0 {
		if err := s.repo.SaveBatch(batch); err != nil {
			return nil, err
		}
	}

	return results, nil
}
