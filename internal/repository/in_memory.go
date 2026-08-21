package repository

import (
	"sync"

	"github.com/Knapptan/Y-URL-shortener/internal/model"
)

type InMemoryRepo struct {
	mu   sync.RWMutex
	data map[string]model.URLRecord
}

func NewInMemoryRepo() *InMemoryRepo {
	return &InMemoryRepo{data: make(map[string]model.URLRecord)}
}

func (r *InMemoryRepo) Save(id string, record model.URLRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.data[id]; exists {
		return ErrIDExists
	}
	r.data[id] = record
	return nil
}

func (r *InMemoryRepo) Get(id string) (model.URLRecord, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.data[id]
	return rec, ok
}
