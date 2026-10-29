package repository

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/Knapptan/Y-URL-shortener/internal/model"
)

// BenchmarkFileRepoGetByUserID измеряет скорость и аллокации GetByUserID.
func BenchmarkFileRepoGetByUserID(b *testing.B) {
	tmp := filepath.Join(b.TempDir(), "storage.json")
	repo, err := NewFileRepository(tmp)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < 100; i++ {
		uid := "u2"
		if i%5 == 0 {
			uid = "u1"
		}
		err := repo.Save(fmt.Sprintf("id-%d", i), model.URLRecord{
			OriginalURL: fmt.Sprintf("https://example.com/%d", i),
			UserID:      uid,
		})
		if err != nil {
			b.Fatal(err)
		}
	}

	for b.Loop() {
		_, _ = repo.GetByUserID("u1")
	}
}

// BenchmarkInMemoryGetByUserID — то же для in-memory репозитория.
func BenchmarkInMemoryGetByUserID(b *testing.B) {
	repo := NewInMemoryRepo()

	for i := 0; i < 100; i++ {
		uid := "u2"
		if i%5 == 0 {
			uid = "u1"
		}
		err := repo.Save(fmt.Sprintf("id-%d", i), model.URLRecord{
			OriginalURL: fmt.Sprintf("https://example.com/%d", i),
			UserID:      uid,
		})
		if err != nil {
			b.Fatal(err)
		}
	}

	for b.Loop() {
		_, _ = repo.GetByUserID("u1")
	}
}
