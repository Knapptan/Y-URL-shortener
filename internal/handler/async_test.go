package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Knapptan/Y-URL-shortener/internal/config"
	"github.com/Knapptan/Y-URL-shortener/internal/model"
	"github.com/Knapptan/Y-URL-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestHandler_AsyncShorten(t *testing.T) {
	t.Run("unauthorized", func(t *testing.T) {
		h := NewURLHandler(service.NewURLService(&mockRepo{}, config.DefaultBaseURL), config.DefaultBaseURL, nil)
		req := httptest.NewRequest(http.MethodPost, "/api/shorten/async", strings.NewReader(`{"url":"https://ya.ru","correlation_id":"corr"}`))
		recorder := httptest.NewRecorder()

		h.AsyncShorten(recorder, req)

		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		h := NewURLHandler(service.NewURLService(&mockRepo{}, config.DefaultBaseURL), config.DefaultBaseURL, nil)
		req := withUserID(httptest.NewRequest(http.MethodPost, "/api/shorten/async", strings.NewReader(`{"url":`)), "user-1")
		recorder := httptest.NewRecorder()

		h.AsyncShorten(recorder, req)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.Equal(t, "Invalid JSON\n", recorder.Body.String())
	})

	t.Run("empty URL", func(t *testing.T) {
		h := NewURLHandler(service.NewURLService(&mockRepo{}, config.DefaultBaseURL), config.DefaultBaseURL, nil)
		req := withUserID(httptest.NewRequest(http.MethodPost, "/api/shorten/async", strings.NewReader(`{"correlation_id":"corr"}`)), "user-1")
		recorder := httptest.NewRecorder()

		h.AsyncShorten(recorder, req)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.Equal(t, "URL is required\n", recorder.Body.String())
	})

	t.Run("empty correlation ID", func(t *testing.T) {
		h := NewURLHandler(service.NewURLService(&mockRepo{}, config.DefaultBaseURL), config.DefaultBaseURL, nil)
		req := withUserID(httptest.NewRequest(http.MethodPost, "/api/shorten/async", strings.NewReader(`{"url":"https://ya.ru"}`)), "user-1")
		recorder := httptest.NewRecorder()

		h.AsyncShorten(recorder, req)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.Equal(t, "correlation_id is required\n", recorder.Body.String())
	})

	t.Run("service error", func(t *testing.T) {
		repo := &mockRepo{
			getByOriginalURLFunc: func(_ string) (string, model.URLRecord, bool, error) {
				return "", model.URLRecord{}, false, errors.New("db error")
			},
		}
		h := NewURLHandler(service.NewURLService(repo, config.DefaultBaseURL), config.DefaultBaseURL, nil)
		req := withUserID(httptest.NewRequest(http.MethodPost, "/api/shorten/async", strings.NewReader(`{"url":"https://ya.ru","correlation_id":"corr"}`)), "user-1")
		recorder := httptest.NewRecorder()

		h.AsyncShorten(recorder, req)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.Equal(t, "db error\n", recorder.Body.String())
	})

	t.Run("new job returns accepted", func(t *testing.T) {
		block := make(chan struct{})
		repo := &mockRepo{
			saveFunc: func(_ string, _ model.URLRecord) error {
				<-block
				return nil
			},
		}
		svc := service.NewURLService(repo, config.DefaultBaseURL)
		h := NewURLHandler(svc, config.DefaultBaseURL, nil)
		req := withUserID(httptest.NewRequest(http.MethodPost, "/api/shorten/async", strings.NewReader(`{"url":"https://ya.ru","correlation_id":"corr"}`)), "user-1")
		recorder := httptest.NewRecorder()

		h.AsyncShorten(recorder, req)

		assert.Equal(t, http.StatusAccepted, recorder.Code)
		var resp asyncShortenResponse
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
		assert.Equal(t, "corr", resp.CorrelationID)
		assert.Equal(t, string(model.JobStatusPending), resp.Status)

		close(block)
		assert.Eventually(t, func() bool {
			job, ok := svc.GetAsyncJob("corr")
			return ok && job.Status == model.JobStatusCompleted
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("existing URL returns completed job", func(t *testing.T) {
		repo := &mockRepo{
			getByOriginalURLFunc: func(_ string) (string, model.URLRecord, bool, error) {
				return "existing", model.URLRecord{OriginalURL: "https://ya.ru"}, true, nil
			},
		}
		h := NewURLHandler(service.NewURLService(repo, config.DefaultBaseURL), config.DefaultBaseURL, nil)
		req := withUserID(httptest.NewRequest(http.MethodPost, "/api/shorten/async", strings.NewReader(`{"url":"https://ya.ru","correlation_id":"corr"}`)), "user-1")
		recorder := httptest.NewRecorder()

		h.AsyncShorten(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		var resp asyncShortenResponse
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
		assert.Equal(t, string(model.JobStatusCompleted), resp.Status)
		require.NotNil(t, resp.Result)
		assert.Equal(t, config.DefaultBaseURL+"/existing", resp.Result.ShortURL)
	})
}

func TestHandler_GetAsyncResult(t *testing.T) {
	t.Run("unauthorized", func(t *testing.T) {
		h := NewURLHandler(service.NewURLService(&mockRepo{}, config.DefaultBaseURL), config.DefaultBaseURL, nil)
		req := withURLParam(httptest.NewRequest(http.MethodGet, "/api/shorten/result/corr", nil), "correlation_id", "corr")
		recorder := httptest.NewRecorder()

		h.GetAsyncResult(recorder, req)

		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	})

	t.Run("missing correlation ID", func(t *testing.T) {
		h := NewURLHandler(service.NewURLService(&mockRepo{}, config.DefaultBaseURL), config.DefaultBaseURL, nil)
		req := withUserID(httptest.NewRequest(http.MethodGet, "/api/shorten/result/", nil), "user-1")
		recorder := httptest.NewRecorder()

		h.GetAsyncResult(recorder, req)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.Equal(t, "correlation_id is required\n", recorder.Body.String())
	})

	t.Run("not found", func(t *testing.T) {
		h := NewURLHandler(service.NewURLService(&mockRepo{}, config.DefaultBaseURL), config.DefaultBaseURL, nil)
		req := withUserID(withURLParam(httptest.NewRequest(http.MethodGet, "/api/shorten/result/missing", nil), "correlation_id", "missing"), "user-1")
		recorder := httptest.NewRecorder()

		h.GetAsyncResult(recorder, req)

		assert.Equal(t, http.StatusNotFound, recorder.Code)
		assert.Equal(t, "correlation_id not found\n", recorder.Body.String())
	})

	t.Run("forbidden for another user", func(t *testing.T) {
		repo := &mockRepo{
			getByOriginalURLFunc: func(_ string) (string, model.URLRecord, bool, error) {
				return "existing", model.URLRecord{OriginalURL: "https://ya.ru"}, true, nil
			},
		}
		svc := service.NewURLService(repo, config.DefaultBaseURL)
		_, _, err := svc.AsyncShorten("https://ya.ru", "corr", "owner")
		require.NoError(t, err)

		h := NewURLHandler(svc, config.DefaultBaseURL, nil)
		req := withUserID(withURLParam(httptest.NewRequest(http.MethodGet, "/api/shorten/result/corr", nil), "correlation_id", "corr"), "other")
		recorder := httptest.NewRecorder()

		h.GetAsyncResult(recorder, req)

		assert.Equal(t, http.StatusForbidden, recorder.Code)
		assert.Equal(t, "Forbidden\n", recorder.Body.String())
	})

	t.Run("completed job", func(t *testing.T) {
		repo := &mockRepo{
			getByOriginalURLFunc: func(_ string) (string, model.URLRecord, bool, error) {
				return "existing", model.URLRecord{OriginalURL: "https://ya.ru"}, true, nil
			},
		}
		svc := service.NewURLService(repo, config.DefaultBaseURL)
		_, _, err := svc.AsyncShorten("https://ya.ru", "corr", "user-1")
		require.NoError(t, err)

		h := NewURLHandler(svc, config.DefaultBaseURL, nil)
		req := withUserID(withURLParam(httptest.NewRequest(http.MethodGet, "/api/shorten/result/corr", nil), "correlation_id", "corr"), "user-1")
		recorder := httptest.NewRecorder()

		h.GetAsyncResult(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		var resp asyncShortenResponse
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
		assert.Equal(t, string(model.JobStatusCompleted), resp.Status)
		require.NotNil(t, resp.Result)
		assert.Equal(t, config.DefaultBaseURL+"/existing", resp.Result.ShortURL)
	})

	t.Run("completed job with error", func(t *testing.T) {
		repo := &mockRepo{
			saveFunc: func(_ string, _ model.URLRecord) error {
				return errors.New("save error")
			},
		}
		svc := service.NewURLService(repo, config.DefaultBaseURL)
		_, _, err := svc.AsyncShorten("https://ya.ru", "corr", "user-1")
		require.NoError(t, err)
		assert.Eventually(t, func() bool {
			job, ok := svc.GetAsyncJob("corr")
			return ok && job.Status == model.JobStatusCompleted && job.Error != ""
		}, time.Second, 10*time.Millisecond)

		h := NewURLHandler(svc, config.DefaultBaseURL, nil)
		req := withUserID(withURLParam(httptest.NewRequest(http.MethodGet, "/api/shorten/result/corr", nil), "correlation_id", "corr"), "user-1")
		recorder := httptest.NewRecorder()

		h.GetAsyncResult(recorder, req)

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		var resp asyncShortenResponse
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
		assert.Equal(t, "save error", resp.Error)
		assert.Nil(t, resp.Result)
	})

	t.Run("pending job", func(t *testing.T) {
		block := make(chan struct{})
		repo := &mockRepo{
			saveFunc: func(_ string, _ model.URLRecord) error {
				<-block
				return nil
			},
		}
		svc := service.NewURLService(repo, config.DefaultBaseURL)
		_, _, err := svc.AsyncShorten("https://ya.ru", "corr", "user-1")
		require.NoError(t, err)

		h := NewURLHandler(svc, config.DefaultBaseURL, nil)
		req := withUserID(withURLParam(httptest.NewRequest(http.MethodGet, "/api/shorten/result/corr", nil), "correlation_id", "corr"), "user-1")
		recorder := httptest.NewRecorder()

		h.GetAsyncResult(recorder, req)

		assert.Equal(t, http.StatusAccepted, recorder.Code)
		var resp asyncShortenResponse
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
		assert.Equal(t, string(model.JobStatusPending), resp.Status)

		close(block)
		assert.Eventually(t, func() bool {
			job, ok := svc.GetAsyncJob("corr")
			return ok && job.Status == model.JobStatusCompleted
		}, time.Second, 10*time.Millisecond)
	})
}
