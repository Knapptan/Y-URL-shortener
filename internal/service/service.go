package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/Knapptan/Y-URL-shortener/internal/model"
	"github.com/Knapptan/Y-URL-shortener/internal/repository"
)

type URLService struct {
	repo repository.URLRepository
}

func NewURLService(repo repository.URLRepository) *URLService {
	return &URLService{repo: repo}
}

// generateShortID генерирует случайный короткий ID длиной 8 символов.
func (s *URLService) generateShortID() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:8], nil
}

func (s *URLService) Shorten(originalURL string) (string, error) {
	originalURL = strings.TrimSpace(originalURL)
	if originalURL == "" {
		return "", errors.New("empty URL")
	}

	var id string
	var err error
	for {
		id, err = s.generateShortID()
		if err != nil {
			return "", err
		}
		record := model.URLRecord{OriginalURL: originalURL}
		err = s.repo.Save(id, record)
		if err == nil {
			break
		}
		if err != repository.ErrIDExists {
			return "", err
		}
		// если ID занят, пробуем сгенерировать новый
	}
	return id, nil
}

func (s *URLService) GetOriginal(id string) (string, bool) {
	if id == "" {
		return "", false
	}
	record, ok := s.repo.Get(id)
	if !ok {
		return "", false
	}
	return record.OriginalURL, true
}
