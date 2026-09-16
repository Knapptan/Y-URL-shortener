package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/Knapptan/Y-URL-shortener/internal/model"
	"github.com/Knapptan/Y-URL-shortener/internal/repository"
)

// URLService предоставляет бизнес-логику для работы с URL.
type URLService struct {
	repo    repository.URLRepository
	baseURL string
}

// BatchItem — элемент пакетного запроса.
type BatchItem struct {
	CorrelationID string
	OriginalURL   string
}

// BatchResult — результат пакетной обработки.
type BatchResult struct {
	CorrelationID string
	ShortURL      string
}

// NewURLService создаёт новый экземпляр URLService.
func NewURLService(repo repository.URLRepository, baseURL string) *URLService {
	return &URLService{repo: repo, baseURL: baseURL}
}

func (s *URLService) generateShortID() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:8], nil
}

// Shorten сокращает URL для указанного пользователя.
func (s *URLService) Shorten(originalURL, userID string) (string, bool, error) {
	originalURL = strings.TrimSpace(originalURL)
	if originalURL == "" {
		return "", false, errors.New("empty URL")
	}

	existingID, _, ok, err := s.repo.GetByOriginalURL(originalURL)
	if err != nil {
		return "", false, err
	}
	if ok {
		return existingID, true, nil
	}

	id, err := s.generateShortID()
	if err != nil {
		return "", false, err
	}
	record := model.URLRecord{OriginalURL: originalURL, UserID: userID}
	if err := s.repo.Save(id, record); err != nil {
		return "", false, err
	}
	return id, false, nil
}

// GetOriginal возвращает оригинальный URL, флаг существования, флаг удаления и ошибку.
func (s *URLService) GetOriginal(id string) (string, bool, bool, error) {
	if id == "" {
		return "", false, false, nil
	}
	record, ok, err := s.repo.Get(id)
	if err != nil {
		return "", false, false, err
	}
	if !ok {
		return "", false, false, nil
	}
	return record.OriginalURL, true, record.Deleted, nil
}

// ShortenBatch обрабатывает пакетный запрос.
func (s *URLService) ShortenBatch(items []BatchItem, userID string) ([]BatchResult, error) {
	if len(items) == 0 {
		return nil, errors.New("empty batch")
	}

	results := make([]BatchResult, 0, len(items))
	batch := make(map[string]model.URLRecord)
	processed := make(map[string]string)

	for _, item := range items {
		if existingID, ok := processed[item.OriginalURL]; ok {
			results = append(results, BatchResult{
				CorrelationID: item.CorrelationID,
				ShortURL:      s.baseURL + "/" + existingID,
			})
			continue
		}

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

		id, err := s.generateShortID()
		if err != nil {
			return nil, err
		}
		batch[id] = model.URLRecord{OriginalURL: item.OriginalURL, UserID: userID}
		processed[item.OriginalURL] = id
		results = append(results, BatchResult{
			CorrelationID: item.CorrelationID,
			ShortURL:      s.baseURL + "/" + id,
		})
	}

	if len(batch) > 0 {
		if err := s.repo.SaveBatch(batch); err != nil {
			return nil, err
		}
	}
	return results, nil
}

// GetUserURLs возвращает все не удалённые URL пользователя.
func (s *URLService) GetUserURLs(userID string) ([]model.URLRecord, error) {
	if userID == "" {
		return nil, errors.New("empty user ID")
	}
	records, err := s.repo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}
	var result []model.URLRecord
	for _, rec := range records {
		if !rec.Deleted {
			result = append(result, rec)
		}
	}
	return result, nil
}

// DeleteURLs помечает URL как удалённые (только принадлежащие пользователю).
func (s *URLService) DeleteURLs(userID string, ids []string) error {
	if userID == "" {
		return errors.New("empty user ID")
	}
	return s.repo.BatchDeleteByUserID(userID, ids)
}
