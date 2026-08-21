package repository

import (
	"errors"

	"github.com/Knapptan/Y-URL-shortener/internal/model"
)

var ErrIDExists = errors.New("ID already exists")

// URLRepository определяет контракт для хранилища URL.
type URLRepository interface {
	Save(id string, record model.URLRecord) error
	Get(id string) (model.URLRecord, bool)
	SaveBatch(batch map[string]model.URLRecord) error
}
