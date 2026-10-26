package repository

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Knapptan/Y-URL-shortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileRepository(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "storage.json")

	// Создаём репозиторий (файла нет – создаётся пустой)
	repo, err := NewFileRepository(filePath)
	require.NoError(t, err)

	// Сохраняем запись
	err = repo.Save("abc", model.URLRecord{OriginalURL: "https://ya.ru"})
	require.NoError(t, err)

	// Проверяем, что файл создался
	_, err = os.Stat(filePath)
	assert.NoError(t, err)

	// Создаём новый репозиторий (загружаем из файла)
	repo2, err := NewFileRepository(filePath)
	require.NoError(t, err)

	// Проверяем, что данные восстановлены
	rec, ok, err := repo2.Get("abc")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "https://ya.ru", rec.OriginalURL)

	// Сохраняем ещё одну запись
	err = repo2.Save("def", model.URLRecord{OriginalURL: "https://google.com"})
	require.NoError(t, err)

	// Проверяем, что старая запись осталась
	rec, ok, err = repo2.Get("abc")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "https://ya.ru", rec.OriginalURL)

	// Тестирование GetByOriginalURL
	t.Run("GetByOriginalURL", func(t *testing.T) {
		// Существующий URL
		id, rec, ok, err := repo2.GetByOriginalURL("https://ya.ru")
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "abc", id)
		assert.Equal(t, "https://ya.ru", rec.OriginalURL)

		// Другой существующий
		id, rec, ok, err = repo2.GetByOriginalURL("https://google.com")
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "def", id)
		assert.Equal(t, "https://google.com", rec.OriginalURL)

		// Несуществующий
		_, _, ok, err = repo2.GetByOriginalURL("https://nonexistent.com")
		require.NoError(t, err)
		assert.False(t, ok)
	})
}
