package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

const CookieName = "user_id"

// GenerateUserID создаёт новый UUID.
func GenerateUserID() string {
	return uuid.New().String()
}

// SignUserID подписывает userID с помощью HMAC-SHA256 и возвращает подписанное значение.
func SignUserID(userID, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(userID))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return userID + ":" + signature
}

// VerifyUserID проверяет подпись и возвращает userID, если подпись верна.
func VerifyUserID(signed, secret string) (string, error) {
	parts := strings.SplitN(signed, ":", 2)
	if len(parts) != 2 {
		return "", errors.New("invalid cookie format")
	}
	userID, signature := parts[0], parts[1]
	if userID == "" {
		return "", errors.New("empty user ID")
	}
	// Вычисляем ожидаемую подпись
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(userID))
	expected := base64.StdEncoding.EncodeToString(h.Sum(nil))
	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return "", errors.New("invalid signature")
	}
	return userID, nil
}

// SetUserCookie создаёт и устанавливает подписанную куку.
func SetUserCookie(w http.ResponseWriter, userID, secret string) {
	signed := SignUserID(userID, secret)
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    signed,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // для локальной разработки, можно true в проде
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((24 * time.Hour).Seconds()), // 1 день
	})
}
