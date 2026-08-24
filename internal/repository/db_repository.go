package repository

import (
	"database/sql"
	"errors"

	"github.com/Knapptan/Y-URL-shortener/internal/model"
	"github.com/lib/pq"
)

// DBRepository реализует URLRepository для хранения данных в PostgreSQL.
type DBRepository struct {
	db *sql.DB
}

// NewDBRepository создаёт новый экземпляр DBRepository.
func NewDBRepository(db *sql.DB) (*DBRepository, error) {
	return &DBRepository{db: db}, nil
}

// Save сохраняет запись по указанному ID.
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

// Get возвращает запись по ID и флаг её существования.
func (r *DBRepository) Get(id string) (model.URLRecord, bool, error) {
	query := `SELECT original_url FROM short_urls WHERE id = $1`
	row := r.db.QueryRow(query, id)
	var originalURL string
	err := row.Scan(&originalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.URLRecord{}, false, nil
		}
		return model.URLRecord{}, false, err
	}
	return model.URLRecord{OriginalURL: originalURL}, true, nil
}

// SaveBatch атомарно сохраняет несколько записей в рамках одной транзакции.
func (r *DBRepository) SaveBatch(batch map[string]model.URLRecord) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // откат при ошибке

	stmt, err := tx.Prepare("INSERT INTO short_urls (id, original_url) VALUES ($1, $2)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for id, rec := range batch {
		_, err := stmt.Exec(id, rec.OriginalURL)
		if err != nil {
			// проверяем на дубликат (код 23505)
			if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
				return ErrIDExists
			}
			return err
		}
	}
	return tx.Commit()
}

// GetByOriginalURL ищет запись по оригинальному URL.
func (r *DBRepository) GetByOriginalURL(originalURL string) (string, model.URLRecord, bool, error) {
	query := `SELECT id, original_url FROM short_urls WHERE original_url = $1`
	row := r.db.QueryRow(query, originalURL)
	var id string
	var orig string
	err := row.Scan(&id, &orig)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", model.URLRecord{}, false, nil
		}
		return "", model.URLRecord{}, false, err
	}
	return id, model.URLRecord{OriginalURL: orig}, true, nil
}
