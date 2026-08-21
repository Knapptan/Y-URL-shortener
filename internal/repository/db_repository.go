package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/Knapptan/Y-URL-shortener/internal/model"
	"github.com/lib/pq"
	_ "github.com/lib/pq" // драйвер PostgreSQL
)

// DBRepository - хранит данные в PostgreSQL.
type DBRepository struct {
	db *sql.DB
}

// NewDBRepository - создаёт репозиторий и выполняет миграцию (создание таблицы).
func NewDBRepository(db *sql.DB) (*DBRepository, error) {
	query := `
        CREATE TABLE IF NOT EXISTS short_urls (
            id TEXT PRIMARY KEY,
            original_url TEXT NOT NULL
        );
    `
	if _, err := db.Exec(query); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}
	return &DBRepository{db: db}, nil
}

// Save - сохраняет ID и оригинальный URL.
func (r *DBRepository) Save(id string, record model.URLRecord) error {
	query := `INSERT INTO short_urls (id, original_url) VALUES ($1, $2)`
	_, err := r.db.Exec(query, id, record.OriginalURL)
	if err != nil {
		// Ошибка дубликата ключа (код 23505 в PostgreSQL)
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return ErrIDExists
		}
		return err
	}
	return nil
}

// Get возвращает оригинальный URL по ID.
func (r *DBRepository) Get(id string) (model.URLRecord, bool) {
	query := `SELECT original_url FROM short_urls WHERE id = $1`
	row := r.db.QueryRow(query, id)
	var originalURL string
	err := row.Scan(&originalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.URLRecord{}, false
		}
		// Логируем ошибку, но возвращаем false
		return model.URLRecord{}, false
	}
	return model.URLRecord{OriginalURL: originalURL}, true
}
