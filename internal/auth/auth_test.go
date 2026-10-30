package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateUserID(t *testing.T) {
	first := GenerateUserID()
	second := GenerateUserID()

	assert.NotEmpty(t, first)
	assert.NotEqual(t, first, second)
	_, err := uuid.Parse(first)
	assert.NoError(t, err)
}

func TestSignAndVerifyUserID(t *testing.T) {
	const secret = "test-secret"
	userID := "user-1"

	signed := SignUserID(userID, secret)

	assert.Contains(t, signed, userID+":")

	verified, err := VerifyUserID(signed, secret)
	require.NoError(t, err)
	assert.Equal(t, userID, verified)
}

func TestVerifyUserIDErrors(t *testing.T) {
	const secret = "test-secret"

	tests := []struct {
		name   string
		signed string
		secret string
	}{
		{name: "invalid format", signed: "no-colon", secret: secret},
		{name: "empty user ID", signed: ":signature", secret: secret},
		{name: "wrong secret", signed: SignUserID("user-1", "other-secret"), secret: secret},
		{name: "invalid signature", signed: "user-1:invalid", secret: secret},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := VerifyUserID(tt.signed, tt.secret)
			assert.Error(t, err)
		})
	}
}

func TestSetUserCookie(t *testing.T) {
	const secret = "test-secret"
	const userID = "user-1"
	recorder := httptest.NewRecorder()

	SetUserCookie(recorder, userID, secret)

	resp := recorder.Result()
	defer resp.Body.Close()
	cookies := resp.Cookies()
	require.Len(t, cookies, 1)

	cookie := cookies[0]
	assert.Equal(t, CookieName, cookie.Name)
	assert.Equal(t, "/", cookie.Path)
	assert.True(t, cookie.HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	assert.Equal(t, int((24 * time.Hour).Seconds()), cookie.MaxAge)

	verified, err := VerifyUserID(cookie.Value, secret)
	require.NoError(t, err)
	assert.Equal(t, userID, verified)
}

func TestSignAndVerify(t *testing.T) {
	secret := "test-secret"
	userID := "user-123"

	signed := SignUserID(userID, secret)
	assert.Contains(t, signed, ":")

	got, err := VerifyUserID(signed, secret)
	require.NoError(t, err)
	assert.Equal(t, userID, got)

	// Неверный secret
	_, err = VerifyUserID(signed, "wrong-secret")
	assert.Error(t, err)

	// Неверный формат
	_, err = VerifyUserID("garbage", secret)
	assert.Error(t, err)

	// Пустой userID
	_, err = VerifyUserID(":signature", secret)
	assert.Error(t, err)
}
