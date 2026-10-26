package repository

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/Knapptan/Y-URL-shortener/internal/model"
)

// FileRepository реализует URLRepository с сохранением в JSON-файл.
type FileRepository struct {
	mu       sync.RWMutex
	data     map[string]model.URLRecord
	filePath string
}

// NewFileRepository создаёт репозиторий и загружает данные из файла.
func NewFileRepository(filePath string) (*FileRepository, error) {
	repo := &FileRepository{
		data:     make(map[string]model.URLRecord),
		filePath: filePath,
	}
	if err := repo.load(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *FileRepository) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	file, err := os.Open(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	var records []model.StorageRecord
	if err := json.NewDecoder(file).Decode(&records); err != nil {
		return err
	}
	for _, rec := range records {
		r.data[rec.ShortURL] = model.URLRecord{
			ID:          rec.ShortURL,
			OriginalURL: rec.OriginalURL,
			UserID:      rec.UserID,
			Deleted:     rec.Deleted,
		}
	}
	return nil
}

func (r *FileRepository) saveToFile() error {
	var records []model.StorageRecord
	for shortURL, rec := range r.data {
		records = append(records, model.StorageRecord{
			UUID:        shortURL,
			ShortURL:    shortURL,
			OriginalURL: rec.OriginalURL,
			UserID:      rec.UserID,
			Deleted:     rec.Deleted,
		})
	}

	file, err := os.Create(r.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(records)
}

// Save сохраняет запись и перезаписывает файл.
func (r *FileRepository) Save(id string, record model.URLRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.data[id]; exists {
		return ErrIDExists
	}
	record.ID = id
	r.data[id] = record
	return r.saveToFile()
}

// Get возвращает запись по ID.
func (r *FileRepository) Get(id string) (model.URLRecord, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.data[id]
	return rec, ok, nil
}

// SaveBatch атомарно сохраняет несколько записей.
func (r *FileRepository) SaveBatch(batch map[string]model.URLRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id := range batch {
		if _, exists := r.data[id]; exists {
			return ErrIDExists
		}
	}
	for id, rec := range batch {
		rec.ID = id
		r.data[id] = rec
	}
	return r.saveToFile()
}

// GetByOriginalURL ищет запись по оригинальному URL.
func (r *FileRepository) GetByOriginalURL(originalURL string) (string, model.URLRecord, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for id, rec := range r.data {
		if rec.OriginalURL == originalURL {
			return id, rec, true, nil
		}
	}
	return "", model.URLRecord{}, false, nil
}

// GetByUserID возвращает все URL пользователя.
func (r *FileRepository) GetByUserID(userID string) ([]model.URLRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []model.URLRecord
	for _, rec := range r.data {
		if rec.UserID == userID {
			result = append(result, rec)
		}
	}
	return result, nil
}

// BatchDeleteByUserID помечает URL пользователя как удалённые.
func (r *FileRepository) BatchDeleteByUserID(userID string, ids []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, id := range ids {
		if rec, ok := r.data[id]; ok && rec.UserID == userID {
			rec.Deleted = true
			r.data[id] = rec
		}
	}
	return r.saveToFile()
}
