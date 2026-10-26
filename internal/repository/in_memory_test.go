package repository

import (
	"testing"

	"github.com/Knapptan/Y-URL-shortener/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestInMemoryRepo(t *testing.T) {
	repo := NewInMemoryRepo()

	// Сохранение новой записи
	err := repo.Save("abc123", model.URLRecord{OriginalURL: "https://ya.ru"})
	assert.NoError(t, err)

	// Попытка сохранить тот же ID – ошибка
	err = repo.Save("abc123", model.URLRecord{OriginalURL: "https://google.com"})
	assert.ErrorIs(t, err, ErrIDExists)

	// Получение существующей записи
	record, ok, err := repo.Get("abc123")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "https://ya.ru", record.OriginalURL)

	// Получение отсутствующей
	_, ok, err = repo.Get("notexist")
	assert.NoError(t, err)
	assert.False(t, ok)

	// Тестирование GetByOriginalURL
	t.Run("GetByOriginalURL", func(t *testing.T) {
		// Существующий URL
		id, rec, ok, err := repo.GetByOriginalURL("https://ya.ru")
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "abc123", id)
		assert.Equal(t, "https://ya.ru", rec.OriginalURL)

		// Несуществующий URL
		_, _, ok, err = repo.GetByOriginalURL("https://nonexistent.com")
		assert.NoError(t, err)
		assert.False(t, ok)

		// Пустая строка
		_, _, ok, err = repo.GetByOriginalURL("")
		assert.NoError(t, err)
		assert.False(t, ok)
	})
}
