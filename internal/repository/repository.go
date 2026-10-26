package repository

import (
	"errors"

	"github.com/Knapptan/Y-URL-shortener/internal/model"
)

var (
	ErrIDExists   = errors.New("ID already exists")
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
