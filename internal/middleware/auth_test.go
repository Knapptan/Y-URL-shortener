package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Knapptan/Y-URL-shortener/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type authResult struct {
	called bool
	userID string
}

func authTestHandler(result *authResult) http.Handler {
	return http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		result.called = true
		result.userID, _ = r.Context().Value(UserIDKey).(string)
	})
}

func TestAuthMiddleware(t *testing.T) {
	const secret = "test-secret"

	t.Run("creates cookie when missing", func(t *testing.T) {
		result := &authResult{}
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)

		AuthMiddleware(secret)(authTestHandler(result)).ServeHTTP(recorder, request)

		assert.True(t, result.called)
		assert.NotEmpty(t, result.userID)

		cookies := recorder.Result().Cookies()
		require.Len(t, cookies, 1)
		verified, err := auth.VerifyUserID(cookies[0].Value, secret)
		require.NoError(t, err)
		assert.Equal(t, result.userID, verified)
	})

	t.Run("uses valid cookie", func(t *testing.T) {
		const userID = "existing-user"
		result := &authResult{}
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.AddCookie(&http.Cookie{
			Name:  auth.CookieName,
			Value: auth.SignUserID(userID, secret),
		})

		AuthMiddleware(secret)(authTestHandler(result)).ServeHTTP(recorder, request)

		assert.True(t, result.called)
		assert.Equal(t, userID, result.userID)
		assert.Empty(t, recorder.Header().Values("Set-Cookie"))
	})

	t.Run("reissues cookie with invalid signature", func(t *testing.T) {
		result := &authResult{}
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.AddCookie(&http.Cookie{
			Name:  auth.CookieName,
			Value: auth.SignUserID("user-1", "wrong-secret"),
		})

		AuthMiddleware(secret)(authTestHandler(result)).ServeHTTP(recorder, request)

		assert.True(t, result.called)
		assert.NotEmpty(t, result.userID)

		cookies := recorder.Result().Cookies()
		require.Len(t, cookies, 1)
		verified, err := auth.VerifyUserID(cookies[0].Value, secret)
		require.NoError(t, err)
		assert.Equal(t, result.userID, verified)
		assert.NotEqual(t, "user-1", result.userID)
	})

	t.Run("reissues cookie with malformed value", func(t *testing.T) {
		result := &authResult{}
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.AddCookie(&http.Cookie{
			Name:  auth.CookieName,
			Value: "malformed",
		})

		AuthMiddleware(secret)(authTestHandler(result)).ServeHTTP(recorder, request)

		assert.True(t, result.called)
		assert.NotEmpty(t, result.userID)
		assert.Len(t, recorder.Result().Cookies(), 1)
	})
}
