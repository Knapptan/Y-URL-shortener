package repository

import (
	"errors"

	"github.com/Knapptan/Y-URL-shortener/internal/model"
)

// Ошибки репозитория.
var (
	// ErrIDExists возвращается при попытке сохранить уже существующий ID.
	ErrIDExists = errors.New("ID already exists")

	// ErrURLDeleted возвращается при обращении к удалённому URL.
	ErrURLDeleted = errors.New("URL is deleted")
)

// URLRepository определяет контракт для хранилища URL.
type URLRepository interface {
	Save(id string, record model.URLRecord) error
	Get(id string) (model.URLRecord, bool, error)
	SaveBatch(batch map[string]model.URLRecord) error
	GetByOriginalURL(originalURL string) (string, model.URLRecord, bool, error)
	GetByUserID(userID string) ([]model.URLRecord, error)
	BatchDeleteByUserID(userID string, ids []string) error
}
