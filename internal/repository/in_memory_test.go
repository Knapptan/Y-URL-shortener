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
	record, ok := repo.Get("abc123")
	assert.True(t, ok)
	assert.Equal(t, "https://ya.ru", record.OriginalURL)

	// Получение отсутствующей
	_, ok = repo.Get("notexist")
	assert.False(t, ok)

	// Тестирование GetByOriginalURL
	t.Run("GetByOriginalURL", func(t *testing.T) {
		// Существующий URL
		id, rec, ok := repo.GetByOriginalURL("https://ya.ru")
		assert.True(t, ok)
		assert.Equal(t, "abc123", id)
		assert.Equal(t, "https://ya.ru", rec.OriginalURL)

		// Несуществующий URL
		_, _, ok = repo.GetByOriginalURL("https://nonexistent.com")
		assert.False(t, ok)

		// Пустая строка
		_, _, ok = repo.GetByOriginalURL("")
		assert.False(t, ok)
	})
}
