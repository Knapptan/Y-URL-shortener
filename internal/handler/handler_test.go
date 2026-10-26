package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Knapptan/Y-URL-shortener/internal/config"
	"github.com/Knapptan/Y-URL-shortener/internal/middleware"
	"github.com/Knapptan/Y-URL-shortener/internal/model"
	"github.com/Knapptan/Y-URL-shortener/internal/service"
	"github.com/stretchr/testify/assert"
)

// mockRepo реализует repository.URLRepository для тестов
type mockRepo struct {
	saveFunc              func(string, model.URLRecord) error
	getFunc               func(string) (model.URLRecord, bool, error)
	saveBatchFunc         func(map[string]model.URLRecord) error
	getByOriginalURLFunc  func(string) (string, model.URLRecord, bool, error)
	getByUserIDFunc       func(string) ([]model.URLRecord, error)
	batchDeleteByUserFunc func(string, []string) error
}

func (m *mockRepo) Save(id string, record model.URLRecord) error {
	return m.saveFunc(id, record)
}

func (m *mockRepo) Get(id string) (model.URLRecord, bool, error) {
	return m.getFunc(id)
}

func (m *mockRepo) SaveBatch(batch map[string]model.URLRecord) error {
	return m.saveBatchFunc(batch)
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
	if m.batchDeleteByUserFunc != nil {
		return m.batchDeleteByUserFunc(userID, ids)
	}
	return nil
}

func withUserID(r *http.Request, userID string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), middleware.UserIDKey, userID))
}

func TestHandler_CreateShortURL(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           string
		mockSave       func(string, model.URLRecord) error
		mockGetByURL   func(string) (string, model.URLRecord, bool, error)
		expectedStatus int
		expectedPrefix string
		expectedBody   string
	}{
		{
			name:   "success",
			method: http.MethodPost,
			body:   "https://ya.ru",
			mockSave: func(_ string, _ model.URLRecord) error {
				return nil
			},
			mockGetByURL: func(_ string) (string, model.URLRecord, bool, error) {
				return "", model.URLRecord{}, false, nil
			},
			expectedStatus: http.StatusCreated,
			expectedPrefix: config.DefaultBaseURL + "/",
		},
		{
			name:           "empty body",
			method:         http.MethodPost,
			body:           "",
			mockSave:       nil,
			mockGetByURL:   nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Empty body\n",
		},
		{
			name:   "repository error",
			method: http.MethodPost,
			body:   "https://ya.ru",
			mockSave: func(_ string, _ model.URLRecord) error {
				return errors.New("some error")
			},
			mockGetByURL: func(_ string) (string, model.URLRecord, bool, error) {
				return "", model.URLRecord{}, false, nil
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "some error\n",
		},
		{
			name:   "conflict",
			method: http.MethodPost,
			body:   "https://ya.ru",
			mockSave: func(_ string, _ model.URLRecord) error {
				return nil
			},
			mockGetByURL: func(_ string) (string, model.URLRecord, bool, error) {
				return "existingID", model.URLRecord{OriginalURL: "https://ya.ru"}, true, nil
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   config.DefaultBaseURL + "/existingID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepo{
				saveFunc:             tt.mockSave,
				getByOriginalURLFunc: tt.mockGetByURL,
			}
			svc := service.NewURLService(repo, config.DefaultBaseURL)
			h := NewURLHandler(svc, config.DefaultBaseURL)

			req := httptest.NewRequest(tt.method, "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()

			h.CreateShortURL(w, withUserID(req, "test-user-123"))

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedPrefix != "" {
				body := w.Body.String()
				assert.True(t, strings.HasPrefix(body, tt.expectedPrefix),
					"ответ должен начинаться с %s, получено %s", tt.expectedPrefix, body)
				id := strings.TrimPrefix(body, tt.expectedPrefix)
				assert.NotEmpty(t, id, "ID не должен быть пустым")
			}
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestHandler_RedirectToOriginal(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		mockGet        func(string) (model.URLRecord, bool, error)
		expectedStatus int
		expectedHeader map[string]string
	}{
		{
			name:   "success redirect",
			method: http.MethodGet,
			path:   "/abc123",
			mockGet: func(id string) (model.URLRecord, bool, error) {
				if id == "abc123" {
					return model.URLRecord{OriginalURL: "https://ya.ru"}, true, nil
				}
				return model.URLRecord{}, false, nil
			},
			expectedStatus: http.StatusTemporaryRedirect,
			expectedHeader: map[string]string{"Location": "https://ya.ru"},
		},
		{
			name:           "missing id",
			method:         http.MethodGet,
			path:           "/",
			mockGet:        nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "not found",
			method: http.MethodGet,
			path:   "/notfound",
			mockGet: func(_ string) (model.URLRecord, bool, error) {
				return model.URLRecord{}, false, nil
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "deleted url returns 410",
			method: http.MethodGet,
			path:   "/deleted123",
			mockGet: func(_ string) (model.URLRecord, bool, error) {
				return model.URLRecord{OriginalURL: "https://ya.ru", Deleted: true}, true, nil
			},
			expectedStatus: http.StatusGone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepo{
				getFunc: tt.mockGet,
			}
			svc := service.NewURLService(repo, config.DefaultBaseURL)
			h := NewURLHandler(svc, config.DefaultBaseURL)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			h.RedirectToOriginal(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			for key, val := range tt.expectedHeader {
				assert.Equal(t, val, w.Header().Get(key))
			}
		})
	}
}

func TestHandler_CreateShortenJSON(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           string
		mockSave       func(string, model.URLRecord) error
		mockGetByURL   func(string) (string, model.URLRecord, bool, error)
		expectedStatus int
		expectedBody   string
		checkJSON      bool
	}{
		{
			name:   "success",
			method: http.MethodPost,
			body:   `{"url":"https://ya.ru"}`,
			mockSave: func(_ string, _ model.URLRecord) error {
				return nil
			},
			mockGetByURL: func(_ string) (string, model.URLRecord, bool, error) {
				return "", model.URLRecord{}, false, nil
			},
			expectedStatus: http.StatusCreated,
			checkJSON:      true,
		},
		{
			name:   "conflict",
			method: http.MethodPost,
			body:   `{"url":"https://ya.ru"}`,
			mockSave: func(_ string, _ model.URLRecord) error {
				return nil
			},
			mockGetByURL: func(_ string) (string, model.URLRecord, bool, error) {
				return "existingID", model.URLRecord{OriginalURL: "https://ya.ru"}, true, nil
			},
			expectedStatus: http.StatusConflict,
			checkJSON:      true,
		},
		{
			name:           "empty url",
			method:         http.MethodPost,
			body:           `{"url":""}`,
			mockSave:       nil,
			mockGetByURL:   nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "URL is empty\n",
		},
		{
			name:           "invalid json",
			method:         http.MethodPost,
			body:           `{"url": "missing quote}`,
			mockSave:       nil,
			mockGetByURL:   nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid JSON\n",
		},
		{
			name:   "repository error",
			method: http.MethodPost,
			body:   `{"url":"https://ya.ru"}`,
			mockSave: func(_ string, _ model.URLRecord) error {
				return errors.New("some repo error")
			},
			mockGetByURL: func(_ string) (string, model.URLRecord, bool, error) {
				return "", model.URLRecord{}, false, nil
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "some repo error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepo{
				saveFunc:             tt.mockSave,
				getByOriginalURLFunc: tt.mockGetByURL,
			}
			svc := service.NewURLService(repo, config.DefaultBaseURL)
			h := NewURLHandler(svc, config.DefaultBaseURL)

			req := httptest.NewRequest(tt.method, "/api/shorten", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// ВАЖНО: вызываем именно CreateShortenJSON, а не CreateShortURL
			h.CreateShortenJSON(w, withUserID(req, "test-user-123"))

			if tt.checkJSON {
				var resp map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				result, ok := resp["result"]
				assert.True(t, ok, "поле result отсутствует")
				assert.True(t, strings.HasPrefix(result, config.DefaultBaseURL+"/"), "неправильный формат URL")
			}

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				if strings.HasPrefix(tt.expectedBody, "{") {
					assert.JSONEq(t, tt.expectedBody, w.Body.String())
				} else {
					assert.Equal(t, tt.expectedBody, w.Body.String())
				}
			}
		})
	}
}

func TestHandler_CreateShortenBatch(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           string
		mockSaveBatch  func(map[string]model.URLRecord) error
		expectedStatus int
		expectedBody   string
		checkJSON      bool
	}{
		{
			name:   "success batch",
			method: http.MethodPost,
			body: `[
				{"correlation_id":"1", "original_url":"https://ya.ru"},
				{"correlation_id":"2", "original_url":"https://google.com"}
			]`,
			mockSaveBatch: func(_ map[string]model.URLRecord) error {
				return nil
			},
			expectedStatus: http.StatusCreated,
			checkJSON:      true,
		},
		{
			name:           "empty batch",
			method:         http.MethodPost,
			body:           `[]`,
			mockSaveBatch:  nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Empty batch\n",
		},
		{
			name:           "invalid json",
			method:         http.MethodPost,
			body:           `[{"correlation_id":"1", "original_url":}`,
			mockSaveBatch:  nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid JSON\n",
		},
		{
			name:   "repository error",
			method: http.MethodPost,
			body: `[
				{"correlation_id":"1", "original_url":"https://ya.ru"}
			]`,
			mockSaveBatch: func(_ map[string]model.URLRecord) error {
				return errors.New("some repo error")
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "some repo error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepo{
				saveBatchFunc: tt.mockSaveBatch,
			}
			svc := service.NewURLService(repo, config.DefaultBaseURL)
			h := NewURLHandler(svc, config.DefaultBaseURL)

			req := httptest.NewRequest(tt.method, "/api/shorten/batch", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// ВАЖНО: вызываем именно CreateShortenBatch
			h.CreateShortenBatch(w, withUserID(req, "test-user-123"))

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkJSON {
				var resp []map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Len(t, resp, 2)
				for _, item := range resp {
					assert.Contains(t, item, "correlation_id")
					assert.Contains(t, item, "short_url")
					assert.True(t, strings.HasPrefix(item["short_url"], config.DefaultBaseURL+"/"))
				}
			}
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestHandler_GetUserURLs(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		mockGetByUser  func(string) ([]model.URLRecord, error)
		expectedStatus int
		checkJSON      bool
		expectedLen    int
	}{
		{
			name:   "success with urls",
			userID: "user-1",
			mockGetByUser: func(_ string) ([]model.URLRecord, error) {
				return []model.URLRecord{
					{ID: "abc", OriginalURL: "https://ya.ru"},
					{ID: "def", OriginalURL: "https://google.com"},
				}, nil
			},
			expectedStatus: http.StatusOK,
			checkJSON:      true,
			expectedLen:    2,
		},
		{
			name:   "no content",
			userID: "user-2",
			mockGetByUser: func(_ string) ([]model.URLRecord, error) {
				return nil, nil
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:   "unauthorized",
			userID: "",
			mockGetByUser: func(_ string) ([]model.URLRecord, error) {
				return nil, nil
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "repository error",
			userID: "user-3",
			mockGetByUser: func(_ string) ([]model.URLRecord, error) {
				return nil, errors.New("db error")
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepo{
				getByUserIDFunc: tt.mockGetByUser,
			}
			svc := service.NewURLService(repo, config.DefaultBaseURL)
			h := NewURLHandler(svc, config.DefaultBaseURL)

			req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
			if tt.userID != "" {
				req = withUserID(req, tt.userID)
			}
			w := httptest.NewRecorder()

			h.GetUserURLs(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkJSON {
				var resp []map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Len(t, resp, tt.expectedLen)
				for _, item := range resp {
					assert.Contains(t, item, "short_url")
					assert.Contains(t, item, "original_url")
					assert.True(t, strings.HasPrefix(item["short_url"], config.DefaultBaseURL+"/"))
				}
			}
		})
	}
}

func TestHandler_DeleteUserURLs(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		body           string
		mockDelete     func(string, []string) error
		expectedStatus int
	}{
		{
			name:   "success",
			userID: "user-1",
			body:   `["id1","id2"]`,
			mockDelete: func(_ string, _ []string) error {
				return nil
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "empty array",
			userID:         "user-1",
			body:           `[]`,
			mockDelete:     nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid json",
			userID:         "user-1",
			body:           `["id1"`,
			mockDelete:     nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "unauthorized",
			userID:         "",
			body:           `["id1"]`,
			mockDelete:     nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "repository error",
			userID: "user-1",
			body:   `["id1"]`,
			mockDelete: func(_ string, _ []string) error {
				return errors.New("db error")
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepo{
				batchDeleteByUserFunc: tt.mockDelete,
			}
			svc := service.NewURLService(repo, config.DefaultBaseURL)
			h := NewURLHandler(svc, config.DefaultBaseURL)

			req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.userID != "" {
				req = withUserID(req, tt.userID)
			}
			w := httptest.NewRecorder()

			h.DeleteUserURLs(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
