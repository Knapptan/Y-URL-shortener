package repository

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
)

// URLRepository интерфейс для работы с хранилищем URL.
type URLRepository interface {
	Save(originalURL string) (string, error) // сохраняет и возвращает ID
	Get(id string) (string, bool)            // получает оригинальный URL по ID
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

	// Генерируем 6 случайных байт -> 8 символов base64
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	id := base64.URLEncoding.EncodeToString(b)[:8]

	// Проверка коллизии
	for {
		if _, exists := r.data[id]; !exists {
			break
		}
		// если вдруг совпало, генерируем новый
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		id = base64.URLEncoding.EncodeToString(b)[:8]
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
