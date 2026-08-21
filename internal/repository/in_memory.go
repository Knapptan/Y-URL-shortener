package repository

import (
	"sync"

	"github.com/Knapptan/Y-URL-shortener/internal/model"
)

// InMemoryRepo реализует URLRepository для хранения данных в оперативной памяти.
type InMemoryRepo struct {
	mu   sync.RWMutex
	data map[string]model.URLRecord
}

// NewInMemoryRepo создаёт новый экземпляр InMemoryRepo с инициализированной картой.
func NewInMemoryRepo() *InMemoryRepo {
	return &InMemoryRepo{data: make(map[string]model.URLRecord)}
}

// Save сохраняет URL-запись по заданному ID.ы
func (r *InMemoryRepo) Save(id string, record model.URLRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.data[id]; exists {
		return ErrIDExists
	}
	r.data[id] = record
	return nil
}

// Get возвращает запись по ID и флаг, указывающий на её существование.
func (r *InMemoryRepo) Get(id string) (model.URLRecord, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.data[id]
	return rec, ok
}

// SaveBatch атомарно сохраняет несколько записей из мапы batch.
func (r *InMemoryRepo) SaveBatch(batch map[string]model.URLRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Проверяем, что все ID свободны
	for id := range batch {
		if _, exists := r.data[id]; exists {
			return ErrIDExists
		}
	}
	// Записываем все записи
	for id, rec := range batch {
		r.data[id] = rec
	}
	return nil
}

// GetByOriginalURL ищет запись по оригинальному URL.
func (r *InMemoryRepo) GetByOriginalURL(originalURL string) (string, model.URLRecord, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for id, rec := range r.data {
		if rec.OriginalURL == originalURL {
			return id, rec, true
		}
	}
	return "", model.URLRecord{}, false
}
