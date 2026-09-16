package repository

import (
	"sync"

	"github.com/Knapptan/Y-URL-shortener/internal/model"
)

// InMemoryRepo реализует URLRepository в оперативной памяти.
type InMemoryRepo struct {
	mu     sync.RWMutex
	store  map[string]model.URLRecord // key = short ID
	urlMap map[string]string          // key = originalURL, value = short ID
}

// NewInMemoryRepo создаёт новое in-memory хранилище.
func NewInMemoryRepo() *InMemoryRepo {
	return &InMemoryRepo{
		store:  make(map[string]model.URLRecord),
		urlMap: make(map[string]string),
	}
}

// Save сохраняет запись.
func (r *InMemoryRepo) Save(id string, record model.URLRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.store[id]; exists {
		return ErrIDExists
	}
	record.ID = id
	r.store[id] = record
	r.urlMap[record.OriginalURL] = id
	return nil
}

// Get возвращает запись по ID.
func (r *InMemoryRepo) Get(id string) (model.URLRecord, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.store[id]
	return rec, ok, nil
}

// SaveBatch атомарно сохраняет несколько записей.
func (r *InMemoryRepo) SaveBatch(batch map[string]model.URLRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id := range batch {
		if _, exists := r.store[id]; exists {
			return ErrIDExists
		}
	}
	for id, rec := range batch {
		rec.ID = id
		r.store[id] = rec
		r.urlMap[rec.OriginalURL] = id
	}
	return nil
}

// GetByOriginalURL ищет запись по оригинальному URL.
func (r *InMemoryRepo) GetByOriginalURL(originalURL string) (string, model.URLRecord, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.urlMap[originalURL]
	if !ok {
		return "", model.URLRecord{}, false, nil
	}
	return id, r.store[id], true, nil
}

// GetByUserID возвращает все URL пользователя.
func (r *InMemoryRepo) GetByUserID(userID string) ([]model.URLRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []model.URLRecord
	for _, rec := range r.store {
		if rec.UserID == userID {
			result = append(result, rec)
		}
	}
	return result, nil
}

// BatchDeleteByUserID помечает URL пользователя как удалённые.
func (r *InMemoryRepo) BatchDeleteByUserID(userID string, ids []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, id := range ids {
		if rec, ok := r.store[id]; ok && rec.UserID == userID {
			rec.Deleted = true
			r.store[id] = rec
		}
	}
	return nil
}
