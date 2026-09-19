package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log/slog"
	"strings"
	"sync"

	"github.com/Knapptan/Y-URL-shortener/internal/model"
	"github.com/Knapptan/Y-URL-shortener/internal/repository"
)

// URLService предоставляет бизнес-логику для работы с URL.
type URLService struct {
	repo    repository.URLRepository
	baseURL string

	// in-memory хранилище асинхронных задач
	jobs   map[string]*model.AsyncJob
	jobsMu sync.RWMutex
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
	return &URLService{
		repo:    repo,
		baseURL: baseURL,
		jobs:    make(map[string]*model.AsyncJob),
	}
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

// GetExistingShortURL возвращает ID уже существующего URL, если он есть.
func (s *URLService) GetExistingShortURL(originalURL string) (string, bool, error) {
	id, _, found, err := s.repo.GetByOriginalURL(originalURL)
	if err != nil {
		return "", false, err
	}
	return id, found, nil
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

// GetUserURLs возвращает все неудалённые URL пользователя.
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

// DeleteURLs помечает URL как удалённые, разбивая список на батчи
// и обрабатывая их параллельно (паттерн fanIn).
func (s *URLService) DeleteURLs(userID string, ids []string) error {
	if userID == "" {
		return errors.New("empty user ID")
	}
	if len(ids) == 0 {
		return nil
	}

	const batchSize = 50
	var wg sync.WaitGroup
	results := make(chan error, (len(ids)/batchSize)+1)

	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[i:end]

		wg.Add(1)
		go func(batchIDs []string) {
			defer wg.Done()
			results <- s.repo.BatchDeleteByUserID(userID, batchIDs)
		}(batch)
	}

	// Закрываем канал после завершения всех горутин
	go func() {
		wg.Wait()
		close(results)
	}()

	// Собираем ошибки из канала
	var firstErr error
	for err := range results {
		if err != nil {
			slog.Error("Batch delete failed", "error", err, "user", userID)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// AsyncShorten создаёт задачу и запускает её обработку в фоне.
// Возвращает копию задачи, чтобы избежать гонок данных с фоновой горутиной.
func (s *URLService) AsyncShorten(url, correlationID, userID string) (*model.AsyncJob, bool, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, false, errors.New("empty URL")
	}
	if correlationID == "" {
		return nil, false, errors.New("empty correlation_id")
	}
	if userID == "" {
		return nil, false, errors.New("empty user ID")
	}

	// Если URL уже есть в хранилище — возвращаем готовую задачу.
	if existingID, found, err := s.GetExistingShortURL(url); err != nil {
		return nil, false, err
	} else if found {
		job := &model.AsyncJob{
			CorrelationID: correlationID,
			URL:           url,
			UserID:        userID,
			ShortID:       existingID,
			ShortURL:      s.baseURL + "/" + existingID,
			Status:        model.JobStatusCompleted,
		}
		s.jobsMu.Lock()
		s.jobs[correlationID] = job
		s.jobsMu.Unlock()

		// Возвращаем копию.
		snapshot := *job
		return &snapshot, true, nil
	}

	// Создаём pending-задачу.
	j := &model.AsyncJob{
		CorrelationID: correlationID,
		URL:           url,
		UserID:        userID,
		Status:        model.JobStatusPending,
	}

	s.jobsMu.Lock()
	s.jobs[correlationID] = j
	s.jobsMu.Unlock()

	// Запускаем фоновую обработку.
	go s.processAsyncJob(j)

	// Возвращаем снимок с гарантированным статусом pending.
	return &model.AsyncJob{
		CorrelationID: correlationID,
		URL:           url,
		UserID:        userID,
		Status:        model.JobStatusPending,
	}, false, nil
}

// processAsyncJob выполняет фактическое сокращение URL в фоне.
func (s *URLService) processAsyncJob(j *model.AsyncJob) {
	id, _, err := s.Shorten(j.URL, j.UserID)
	if err != nil {
		slog.Error("Async shorten failed", "correlation_id", j.CorrelationID, "error", err)
		s.jobsMu.Lock()
		j.Status = model.JobStatusCompleted
		j.Error = err.Error()
		s.jobsMu.Unlock()
		return
	}

	s.jobsMu.Lock()
	j.ShortID = id
	j.ShortURL = s.baseURL + "/" + id
	j.Status = model.JobStatusCompleted
	s.jobsMu.Unlock()
}

// GetAsyncJob возвращает копию задачи по correlation_id.
func (s *URLService) GetAsyncJob(correlationID string) (*model.AsyncJob, bool) {
	s.jobsMu.RLock()
	defer s.jobsMu.RUnlock()

	job, ok := s.jobs[correlationID]
	if !ok {
		return nil, false
	}
	// Возвращаем копию, чтобы избежать гонок данных
	copyJob := *job
	return &copyJob, true
}
