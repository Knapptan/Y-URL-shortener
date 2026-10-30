package service_test

import (
	"fmt"

	"github.com/Knapptan/Y-URL-shortener/internal/model"
	"github.com/Knapptan/Y-URL-shortener/internal/service"
)

// exampleRepo — детерминированный репозиторий для примеров.
type exampleRepo struct {
	urls map[string]model.URLRecord
}

func newExampleRepo() *exampleRepo {
	return &exampleRepo{urls: make(map[string]model.URLRecord)}
}

func (r *exampleRepo) Save(id string, record model.URLRecord) error {
	record.ID = id
	r.urls[id] = record
	return nil
}

func (r *exampleRepo) Get(id string) (model.URLRecord, bool, error) {
	rec, ok := r.urls[id]
	return rec, ok, nil
}

func (r *exampleRepo) SaveBatch(batch map[string]model.URLRecord) error {
	for id, rec := range batch {
		rec.ID = id
		r.urls[id] = rec
	}
	return nil
}

func (r *exampleRepo) GetByOriginalURL(originalURL string) (string, model.URLRecord, bool, error) {
	for id, rec := range r.urls {
		if rec.OriginalURL == originalURL {
			return id, rec, true, nil
		}
	}
	return "", model.URLRecord{}, false, nil
}

func (r *exampleRepo) GetByUserID(userID string) ([]model.URLRecord, error) {
	return nil, nil
}

func (r *exampleRepo) BatchDeleteByUserID(userID string, ids []string) error {
	return nil
}

// ExampleURLService_GetOriginal демонстрирует получение оригинального URL по ID.
func ExampleURLService_GetOriginal() {
	repo := newExampleRepo()
	_ = repo.Save("abc123", model.URLRecord{OriginalURL: "https://practicum.yandex.ru"})

	svc := service.NewURLService(repo, "http://localhost:8080")

	url, exists, deleted, err := svc.GetOriginal("abc123")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("exists:", exists)
	fmt.Println("deleted:", deleted)
	fmt.Println("url:", url)
	// Output:
	// exists: true
	// deleted: false
	// url: https://practicum.yandex.ru
}

// ExampleURLService_GetOriginal_notFound демонстрирует случай, когда URL не найден.
func ExampleURLService_GetOriginal_notFound() {
	repo := newExampleRepo()
	svc := service.NewURLService(repo, "http://localhost:8080")

	_, exists, _, err := svc.GetOriginal("missing")
	fmt.Println("exists:", exists)
	fmt.Println("error:", err)
	// Output:
	// exists: false
	// error: <nil>
}
