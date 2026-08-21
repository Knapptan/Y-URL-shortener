package repository

import (
	"database/sql"
	"os"
	"testing"

	"github.com/Knapptan/Y-URL-shortener/internal/model"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBRepository(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping DB tests")
	}

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()

	// Очищаем таблицу перед тестом (чтобы не мешали старые данные)
	_, _ = db.Exec("DELETE FROM short_urls")

	repo, err := NewDBRepository(db)
	require.NoError(t, err)

	// Сохраняем запись
	err = repo.Save("test123", model.URLRecord{OriginalURL: "https://example.com"})
	assert.NoError(t, err)

	// Попытка сохранить дубликат – ошибка
	err = repo.Save("test123", model.URLRecord{OriginalURL: "https://example2.com"})
	assert.ErrorIs(t, err, ErrIDExists)

	// Получение существующей записи
	record, ok := repo.Get("test123")
	assert.True(t, ok)
	assert.Equal(t, "https://example.com", record.OriginalURL)

	// Получение отсутствующей
	_, ok = repo.Get("notexist")
	assert.False(t, ok)

	// Проверка миграции (таблица создана)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM short_urls").Scan(&count)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, count, 1)
}
