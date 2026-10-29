package service

import (
	"fmt"
	"testing"

	"github.com/Knapptan/Y-URL-shortener/internal/model"
)

// benchRepo — минимальная реализация URLRepository для бенчмарков.
type benchRepo struct {
	data map[string]model.URLRecord
}

func newBenchRepo() *benchRepo {
	return &benchRepo{data: make(map[string]model.URLRecord)}
}

func (r *benchRepo) Save(id string, record model.URLRecord) error {
	r.data[id] = record
	return nil
}

func (r *benchRepo) Get(id string) (model.URLRecord, bool, error) {
	rec, ok := r.data[id]
	return rec, ok, nil
}

func (r *benchRepo) SaveBatch(batch map[string]model.URLRecord) error {
	for id, rec := range batch {
		r.data[id] = rec
	}
	return nil
}

func (r *benchRepo) GetByOriginalURL(originalURL string) (string, model.URLRecord, bool, error) {
	for id, rec := range r.data {
		if rec.OriginalURL == originalURL {
			return id, rec, true, nil
		}
	}
	return "", model.URLRecord{}, false, nil
}

func (r *benchRepo) GetByUserID(userID string) ([]model.URLRecord, error) {
	var out []model.URLRecord
	for _, rec := range r.data {
		if rec.UserID == userID {
			out = append(out, rec)
		}
	}
	return out, nil
}

func (r *benchRepo) BatchDeleteByUserID(userID string, ids []string) error {
	return nil
}

// BenchmarkShorten измеряет скорость создания короткого URL.
func BenchmarkShorten(b *testing.B) {
	repo := newBenchRepo()
	svc := NewURLService(repo, "http://localhost:8080")
	userID := "bench-user"

	i := 0
	for b.Loop() {
		url := fmt.Sprintf("https://example.com/very/long/path/%d", i)
		i++
		if _, _, err := svc.Shorten(url, userID); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGetOriginal измеряет скорость получения оригинального URL.
func BenchmarkGetOriginal(b *testing.B) {
	repo := newBenchRepo()
	svc := NewURLService(repo, "http://localhost:8080")
	id, _, err := svc.Shorten("https://example.com", "bench-user")
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		_, _, _, _ = svc.GetOriginal(id)
	}
}
