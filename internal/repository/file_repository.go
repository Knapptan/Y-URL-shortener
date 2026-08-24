package repository

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/Knapptan/Y-URL-shortener/internal/model"
)

// FileRepository реализует URLRepository с сохранением данных в файл в формате JSON.
type FileRepository struct {
	mu       sync.RWMutex
	data     map[string]model.URLRecord
	filePath string
}

// NewFileRepository создаёт новый экземпляр FileRepository.
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

// load загружает данные из файла в память.
func (r *FileRepository) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	file, err := os.Open(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // файла нет – начинаем с пустой мапы
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
			OriginalURL: rec.OriginalURL,
		}
	}
	return nil
}

// saveToFile записывает все данные из памяти в файл.
func (r *FileRepository) saveToFile() error {
	var records []model.StorageRecord
	for shortURL, rec := range r.data {
		records = append(records, model.StorageRecord{
			UUID:        shortURL,
			ShortURL:    shortURL,
			OriginalURL: rec.OriginalURL,
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

// Save сохраняет запись по указанному ID.
func (r *FileRepository) Save(id string, record model.URLRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.data[id]; exists {
		return ErrIDExists
	}
	r.data[id] = record

	return r.saveToFile()
}

// Get возвращает запись по ID и флаг её существования.
func (r *FileRepository) Get(id string) (model.URLRecord, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.data[id]
	return rec, ok, nil
}

// SaveBatch атомарно сохраняет несколько записей из мапы batch.
func (r *FileRepository) SaveBatch(batch map[string]model.URLRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id := range batch {
		if _, exists := r.data[id]; exists {
			return ErrIDExists
		}
	}
	for id, rec := range batch {
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
