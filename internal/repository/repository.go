package repository

import (
	"crypto/rand"
	"encoding/base64"
	"sync"

	"github.com/Knapptan/Y-URL-shortener/internal/model"
)

// URLRepository интерфейс для работы с хранилищем URL.
type URLRepository interface {
	Save(id string, record model.URLRecord) error
	Get(id string) (model.URLRecord, bool)
}

// InMemoryRepo реализация в памяти.
type InMemoryRepo struct {
	mu   sync.RWMutex
	data map[string]string // key = shortID, value = originalURL
}

// NewInMemoryRepo конструктор.
func NewInMemoryRepo() *InMemoryRepo {
	return &InMemoryRepo{
		data: make(map[string]string),
	}
}

// Save генерирует короткий ID и сохраняет URL.
func (r *InMemoryRepo) Save(originalURL string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var id string

	for {
		// генерируем случайный ID
		b := make([]byte, 6)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		id = base64.URLEncoding.EncodeToString(b)[:8]

		// проверяем, не занят ли ID
		if _, exists := r.data[id]; !exists {
			break
		}
		// если занят – цикл повторится с новой генерацией
	}

	r.data[id] = originalURL
	return id, nil
}

// Get возвращает оригинальный URL по ID.
func (r *InMemoryRepo) Get(id string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	val, ok := r.data[id]
	return val, ok
}
