package service

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Knapptan/Y-URL-shortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	saveFunc             func(string, model.URLRecord) error
	getFunc              func(string) (model.URLRecord, bool, error)
	saveBatchFunc        func(map[string]model.URLRecord) error
	getByOriginalURLFunc func(string) (string, model.URLRecord, bool, error)
	getByUserIDFunc      func(string) ([]model.URLRecord, error)
	batchDeleteFunc      func(string, []string) error
}

func (m *mockRepo) Save(id string, record model.URLRecord) error {
	if m.saveFunc != nil {
		return m.saveFunc(id, record)
	}
	return nil
}

func (m *mockRepo) Get(id string) (model.URLRecord, bool, error) {
	if m.getFunc != nil {
		return m.getFunc(id)
	}
	return model.URLRecord{}, false, nil
}

func (m *mockRepo) SaveBatch(batch map[string]model.URLRecord) error {
	if m.saveBatchFunc != nil {
		return m.saveBatchFunc(batch)
	}
	return nil
}

func (m *mockRepo) GetByOriginalURL(originalURL string) (string, model.URLRecord, bool, error) {
	if m.getByOriginalURLFunc != nil {
		return m.getByOriginalURLFunc(originalURL)
	}
	return "", model.URLRecord{}, false, nil
}

func (m *mockRepo) GetByUserID(userID string) ([]model.URLRecord, error) {
	if m.getByUserIDFunc != nil {
		return m.getByUserIDFunc(userID)
	}
	return nil, nil
}

func (m *mockRepo) BatchDeleteByUserID(userID string, ids []string) error {
	if m.batchDeleteFunc != nil {
		return m.batchDeleteFunc(userID, ids)
	}
	return nil
}

func TestShorten(t *testing.T) {
	t.Run("new URL", func(t *testing.T) {
		var savedID string
		var savedRecord model.URLRecord
		repo := &mockRepo{
			saveFunc: func(id string, record model.URLRecord) error {
				savedID = id
				savedRecord = record
				return nil
			},
		}
		svc := NewURLService(repo, "http://short")

		id, exists, err := svc.Shorten(" https://ya.ru ", "user-1")

		require.NoError(t, err)
		assert.False(t, exists)
		assert.NotEmpty(t, id)
		assert.Equal(t, id, savedID)
		assert.Equal(t, "https://ya.ru", savedRecord.OriginalURL)
		assert.Equal(t, "user-1", savedRecord.UserID)
	})

	t.Run("existing URL", func(t *testing.T) {
		repo := &mockRepo{
			getByOriginalURLFunc: func(_ string) (string, model.URLRecord, bool, error) {
				return "existing", model.URLRecord{OriginalURL: "https://ya.ru"}, true, nil
			},
			saveFunc: func(_ string, _ model.URLRecord) error {
				t.Fatal("Save should not be called for existing URL")
				return nil
			},
		}
		svc := NewURLService(repo, "http://short")

		id, exists, err := svc.Shorten("https://ya.ru", "user-1")

		require.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, "existing", id)
	})

	t.Run("empty URL", func(t *testing.T) {
		svc := NewURLService(&mockRepo{}, "http://short")

		_, _, err := svc.Shorten("   ", "user-1")

		assert.Error(t, err)
	})

	t.Run("lookup error", func(t *testing.T) {
		repoErr := errors.New("db error")
		repo := &mockRepo{
			getByOriginalURLFunc: func(_ string) (string, model.URLRecord, bool, error) {
				return "", model.URLRecord{}, false, repoErr
			},
		}
		svc := NewURLService(repo, "http://short")

		_, _, err := svc.Shorten("https://ya.ru", "user-1")

		assert.ErrorIs(t, err, repoErr)
	})

	t.Run("save error", func(t *testing.T) {
		repoErr := errors.New("save error")
		repo := &mockRepo{
			saveFunc: func(_ string, _ model.URLRecord) error {
				return repoErr
			},
		}
		svc := NewURLService(repo, "http://short")

		_, _, err := svc.Shorten("https://ya.ru", "user-1")

		assert.ErrorIs(t, err, repoErr)
	})
}

func TestGetExistingShortURL(t *testing.T) {
	repoErr := errors.New("db error")

	tests := []struct {
		name        string
		repo        *mockRepo
		expectedID  string
		expectedOK  bool
		expectedErr error
	}{
		{
			name: "found",
			repo: &mockRepo{
				getByOriginalURLFunc: func(_ string) (string, model.URLRecord, bool, error) {
					return "abc", model.URLRecord{OriginalURL: "https://ya.ru"}, true, nil
				},
			},
			expectedID: "abc",
			expectedOK: true,
		},
		{
			name: "not found",
			repo: &mockRepo{
				getByOriginalURLFunc: func(_ string) (string, model.URLRecord, bool, error) {
					return "", model.URLRecord{}, false, nil
				},
			},
		},
		{
			name: "error",
			repo: &mockRepo{
				getByOriginalURLFunc: func(_ string) (string, model.URLRecord, bool, error) {
					return "", model.URLRecord{}, false, repoErr
				},
			},
			expectedErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewURLService(tt.repo, "http://short")

			id, ok, err := svc.GetExistingShortURL("https://ya.ru")

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedID, id)
			assert.Equal(t, tt.expectedOK, ok)
		})
	}
}

func TestGetOriginal(t *testing.T) {
	repoErr := errors.New("db error")

	tests := []struct {
		name            string
		id              string
		repo            *mockRepo
		expectedURL     string
		expectedOK      bool
		expectedDeleted bool
		expectedErr     error
	}{
		{
			name: "empty ID",
			id:   "",
			repo: &mockRepo{},
		},
		{
			name: "found",
			id:   "abc",
			repo: &mockRepo{
				getFunc: func(_ string) (model.URLRecord, bool, error) {
					return model.URLRecord{OriginalURL: "https://ya.ru"}, true, nil
				},
			},
			expectedURL: "https://ya.ru",
			expectedOK:  true,
		},
		{
			name: "deleted",
			id:   "abc",
			repo: &mockRepo{
				getFunc: func(_ string) (model.URLRecord, bool, error) {
					return model.URLRecord{OriginalURL: "https://ya.ru", Deleted: true}, true, nil
				},
			},
			expectedURL:     "https://ya.ru",
			expectedOK:      true,
			expectedDeleted: true,
		},
		{
			name: "not found",
			id:   "abc",
			repo: &mockRepo{
				getFunc: func(_ string) (model.URLRecord, bool, error) {
					return model.URLRecord{}, false, nil
				},
			},
		},
		{
			name: "repo error",
			id:   "abc",
			repo: &mockRepo{
				getFunc: func(_ string) (model.URLRecord, bool, error) {
					return model.URLRecord{}, false, repoErr
				},
			},
			expectedErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewURLService(tt.repo, "http://short")

			url, ok, deleted, err := svc.GetOriginal(tt.id)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedURL, url)
			assert.Equal(t, tt.expectedOK, ok)
			assert.Equal(t, tt.expectedDeleted, deleted)
		})
	}
}

func TestShortenBatch(t *testing.T) {
	t.Run("empty batch", func(t *testing.T) {
		svc := NewURLService(&mockRepo{}, "http://short")

		_, err := svc.ShortenBatch(nil, "user-1")

		assert.Error(t, err)
	})

	t.Run("new URLs", func(t *testing.T) {
		var savedBatch map[string]model.URLRecord
		repo := &mockRepo{
			saveBatchFunc: func(batch map[string]model.URLRecord) error {
				savedBatch = make(map[string]model.URLRecord, len(batch))
				for id, rec := range batch {
					savedBatch[id] = rec
				}
				return nil
			},
		}
		svc := NewURLService(repo, "http://short")
		items := []BatchItem{
			{CorrelationID: "1", OriginalURL: "https://ya.ru"},
			{CorrelationID: "2", OriginalURL: "https://google.com"},
		}

		results, err := svc.ShortenBatch(items, "user-1")

		require.NoError(t, err)
		require.Len(t, results, 2)
		assert.Len(t, savedBatch, 2)
		for i, result := range results {
			assert.Equal(t, items[i].CorrelationID, result.CorrelationID)
			assert.True(t, strings.HasPrefix(result.ShortURL, "http://short/"))
			id := strings.TrimPrefix(result.ShortURL, "http://short/")
			rec, ok := savedBatch[id]
			require.True(t, ok)
			assert.Equal(t, items[i].OriginalURL, rec.OriginalURL)
			assert.Equal(t, "user-1", rec.UserID)
		}
	})

	t.Run("duplicate URLs in batch", func(t *testing.T) {
		var savedBatch map[string]model.URLRecord
		repo := &mockRepo{
			saveBatchFunc: func(batch map[string]model.URLRecord) error {
				savedBatch = make(map[string]model.URLRecord, len(batch))
				for id, rec := range batch {
					savedBatch[id] = rec
				}
				return nil
			},
		}
		svc := NewURLService(repo, "http://short")
		items := []BatchItem{
			{CorrelationID: "1", OriginalURL: "https://ya.ru"},
			{CorrelationID: "2", OriginalURL: "https://ya.ru"},
		}

		results, err := svc.ShortenBatch(items, "user-1")

		require.NoError(t, err)
		require.Len(t, results, 2)
		assert.Len(t, savedBatch, 1)
		assert.Equal(t, results[0].ShortURL, results[1].ShortURL)
	})

	t.Run("existing URL", func(t *testing.T) {
		var savedBatch map[string]model.URLRecord
		repo := &mockRepo{
			getByOriginalURLFunc: func(originalURL string) (string, model.URLRecord, bool, error) {
				if originalURL == "https://ya.ru" {
					return "existing", model.URLRecord{OriginalURL: originalURL}, true, nil
				}
				return "", model.URLRecord{}, false, nil
			},
			saveBatchFunc: func(batch map[string]model.URLRecord) error {
				savedBatch = make(map[string]model.URLRecord, len(batch))
				for id, rec := range batch {
					savedBatch[id] = rec
				}
				return nil
			},
		}
		svc := NewURLService(repo, "http://short")
		items := []BatchItem{
			{CorrelationID: "1", OriginalURL: "https://ya.ru"},
			{CorrelationID: "2", OriginalURL: "https://google.com"},
		}

		results, err := svc.ShortenBatch(items, "user-1")

		require.NoError(t, err)
		require.Len(t, results, 2)
		assert.Equal(t, "http://short/existing", results[0].ShortURL)
		assert.Len(t, savedBatch, 1)
	})

	t.Run("lookup error", func(t *testing.T) {
		repoErr := errors.New("db error")
		repo := &mockRepo{
			getByOriginalURLFunc: func(_ string) (string, model.URLRecord, bool, error) {
				return "", model.URLRecord{}, false, repoErr
			},
		}
		svc := NewURLService(repo, "http://short")

		_, err := svc.ShortenBatch([]BatchItem{{CorrelationID: "1", OriginalURL: "https://ya.ru"}}, "user-1")

		assert.ErrorIs(t, err, repoErr)
	})

	t.Run("save batch error", func(t *testing.T) {
		repoErr := errors.New("save error")
		repo := &mockRepo{
			saveBatchFunc: func(_ map[string]model.URLRecord) error {
				return repoErr
			},
		}
		svc := NewURLService(repo, "http://short")

		_, err := svc.ShortenBatch([]BatchItem{{CorrelationID: "1", OriginalURL: "https://ya.ru"}}, "user-1")

		assert.ErrorIs(t, err, repoErr)
	})
}

func TestGetUserURLs(t *testing.T) {
	repoErr := errors.New("db error")

	t.Run("empty user ID", func(t *testing.T) {
		svc := NewURLService(&mockRepo{}, "http://short")

		_, err := svc.GetUserURLs("")

		assert.Error(t, err)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &mockRepo{
			getByUserIDFunc: func(_ string) ([]model.URLRecord, error) {
				return nil, repoErr
			},
		}
		svc := NewURLService(repo, "http://short")

		_, err := svc.GetUserURLs("user-1")

		assert.ErrorIs(t, err, repoErr)
	})

	t.Run("filters deleted URLs", func(t *testing.T) {
		repo := &mockRepo{
			getByUserIDFunc: func(_ string) ([]model.URLRecord, error) {
				return []model.URLRecord{
					{ID: "1", OriginalURL: "https://ya.ru"},
					{ID: "2", OriginalURL: "https://google.com", Deleted: true},
					{ID: "3", OriginalURL: "https://example.com"},
				}, nil
			},
		}
		svc := NewURLService(repo, "http://short")

		records, err := svc.GetUserURLs("user-1")

		require.NoError(t, err)
		require.Len(t, records, 2)
		assert.Equal(t, "1", records[0].ID)
		assert.Equal(t, "3", records[1].ID)
	})

	t.Run("no URLs", func(t *testing.T) {
		svc := NewURLService(&mockRepo{}, "http://short")

		records, err := svc.GetUserURLs("user-1")

		require.NoError(t, err)
		assert.Empty(t, records)
	})
}

func TestDeleteURLs(t *testing.T) {
	t.Run("empty user ID", func(t *testing.T) {
		var called bool
		repo := &mockRepo{
			batchDeleteFunc: func(_ string, _ []string) error {
				called = true
				return nil
			},
		}
		svc := NewURLService(repo, "http://short")

		err := svc.DeleteURLs("", []string{"1"})

		assert.Error(t, err)
		assert.False(t, called)
	})

	t.Run("empty IDs", func(t *testing.T) {
		var called bool
		repo := &mockRepo{
			batchDeleteFunc: func(_ string, _ []string) error {
				called = true
				return nil
			},
		}
		svc := NewURLService(repo, "http://short")

		err := svc.DeleteURLs("user-1", nil)

		assert.NoError(t, err)
		assert.False(t, called)
	})

	t.Run("splits into batches", func(t *testing.T) {
		var mu sync.Mutex
		var calls [][]string
		repo := &mockRepo{
			batchDeleteFunc: func(_ string, ids []string) error {
				batch := append([]string(nil), ids...)
				mu.Lock()
				calls = append(calls, batch)
				mu.Unlock()
				return nil
			},
		}
		svc := NewURLService(repo, "http://short")
		ids := make([]string, 120)
		for i := range ids {
			ids[i] = "id"
		}

		err := svc.DeleteURLs("user-1", ids)

		require.NoError(t, err)
		mu.Lock()
		defer mu.Unlock()
		assert.Len(t, calls, 3)
		total := 0
		for _, batch := range calls {
			total += len(batch)
		}
		assert.Equal(t, 120, total)
	})

	t.Run("returns first error", func(t *testing.T) {
		repoErr := errors.New("delete error")
		repo := &mockRepo{
			batchDeleteFunc: func(_ string, _ []string) error {
				return repoErr
			},
		}
		svc := NewURLService(repo, "http://short")

		err := svc.DeleteURLs("user-1", []string{"1"})

		assert.ErrorIs(t, err, repoErr)
	})
}

func TestAsyncShorten(t *testing.T) {
	t.Run("validation errors", func(t *testing.T) {
		tests := []struct {
			name          string
			url           string
			correlationID string
			userID        string
		}{
			{name: "empty URL", correlationID: "corr", userID: "user-1"},
			{name: "empty correlation ID", url: "https://ya.ru", userID: "user-1"},
			{name: "empty user ID", url: "https://ya.ru", correlationID: "corr"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				svc := NewURLService(&mockRepo{}, "http://short")

				_, _, err := svc.AsyncShorten(tt.url, tt.correlationID, tt.userID)

				assert.Error(t, err)
			})
		}
	})

	t.Run("existing URL", func(t *testing.T) {
		repo := &mockRepo{
			getByOriginalURLFunc: func(_ string) (string, model.URLRecord, bool, error) {
				return "existing", model.URLRecord{OriginalURL: "https://ya.ru"}, true, nil
			},
		}
		svc := NewURLService(repo, "http://short")

		job, alreadyExists, err := svc.AsyncShorten("https://ya.ru", "corr", "user-1")

		require.NoError(t, err)
		assert.True(t, alreadyExists)
		assert.Equal(t, model.JobStatusCompleted, job.Status)
		assert.Equal(t, "http://short/existing", job.ShortURL)

		stored, ok := svc.GetAsyncJob("corr")
		require.True(t, ok)
		assert.Equal(t, model.JobStatusCompleted, stored.Status)
	})

	t.Run("lookup error", func(t *testing.T) {
		repoErr := errors.New("db error")
		repo := &mockRepo{
			getByOriginalURLFunc: func(_ string) (string, model.URLRecord, bool, error) {
				return "", model.URLRecord{}, false, repoErr
			},
		}
		svc := NewURLService(repo, "http://short")

		_, _, err := svc.AsyncShorten("https://ya.ru", "corr", "user-1")

		assert.ErrorIs(t, err, repoErr)
	})

	t.Run("new URL completes in background", func(t *testing.T) {
		repo := &mockRepo{}
		svc := NewURLService(repo, "http://short")

		job, alreadyExists, err := svc.AsyncShorten("https://ya.ru", "corr", "user-1")

		require.NoError(t, err)
		assert.False(t, alreadyExists)
		assert.Equal(t, model.JobStatusPending, job.Status)

		assert.Eventually(t, func() bool {
			stored, ok := svc.GetAsyncJob("corr")
			return ok && stored.Status == model.JobStatusCompleted && stored.ShortURL != ""
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("background error is saved", func(t *testing.T) {
		repo := &mockRepo{
			saveFunc: func(_ string, _ model.URLRecord) error {
				return errors.New("save error")
			},
		}
		svc := NewURLService(repo, "http://short")

		_, _, err := svc.AsyncShorten("https://ya.ru", "corr", "user-1")
		require.NoError(t, err)

		assert.Eventually(t, func() bool {
			stored, ok := svc.GetAsyncJob("corr")
			return ok && stored.Status == model.JobStatusCompleted && stored.Error != ""
		}, time.Second, 10*time.Millisecond)
	})
}

func TestGetAsyncJob(t *testing.T) {
	svc := NewURLService(&mockRepo{}, "http://short")

	_, ok := svc.GetAsyncJob("missing")

	assert.False(t, ok)
}
