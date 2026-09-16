package repository

import (
	"database/sql"
	"errors"

	"github.com/Knapptan/Y-URL-shortener/internal/model"
	"github.com/lib/pq"
)

// DBRepository реализует URLRepository для PostgreSQL.
type DBRepository struct {
	db *sql.DB
}

// NewDBRepository создаёт репозиторий (схема создаётся миграциями).
func NewDBRepository(db *sql.DB) (*DBRepository, error) {
	return &DBRepository{db: db}, nil
}

// Save сохраняет запись.
func (r *DBRepository) Save(id string, record model.URLRecord) error {
	query := `INSERT INTO short_urls (id, original_url, user_id, is_deleted) VALUES ($1, $2, $3, $4)`
	_, err := r.db.Exec(query, id, record.OriginalURL, record.UserID, record.Deleted)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return ErrIDExists
		}
		return err
	}
	return nil
}

// Get возвращает запись по ID.
func (r *DBRepository) Get(id string) (model.URLRecord, bool, error) {
	query := `SELECT id, original_url, user_id, is_deleted FROM short_urls WHERE id = $1`
	var rec model.URLRecord
	err := r.db.QueryRow(query, id).Scan(&rec.ID, &rec.OriginalURL, &rec.UserID, &rec.Deleted)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.URLRecord{}, false, nil
		}
		return model.URLRecord{}, false, err
	}
	return rec, true, nil
}

// SaveBatch атомарно сохраняет несколько записей.
func (r *DBRepository) SaveBatch(batch map[string]model.URLRecord) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO short_urls (id, original_url, user_id, is_deleted) VALUES ($1, $2, $3, $4)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for id, rec := range batch {
		_, err := stmt.Exec(id, rec.OriginalURL, rec.UserID, rec.Deleted)
		if err != nil {
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
	query := `SELECT id, original_url, user_id, is_deleted FROM short_urls WHERE original_url = $1`
	var rec model.URLRecord
	err := r.db.QueryRow(query, originalURL).Scan(&rec.ID, &rec.OriginalURL, &rec.UserID, &rec.Deleted)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", model.URLRecord{}, false, nil
		}
		return "", model.URLRecord{}, false, err
	}
	return rec.ID, rec, true, nil
}

// GetByUserID возвращает все URL пользователя.
func (r *DBRepository) GetByUserID(userID string) ([]model.URLRecord, error) {
	query := `SELECT id, original_url, user_id, is_deleted FROM short_urls WHERE user_id = $1`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []model.URLRecord
	for rows.Next() {
		var rec model.URLRecord
		if err := rows.Scan(&rec.ID, &rec.OriginalURL, &rec.UserID, &rec.Deleted); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// BatchDeleteByUserID помечает URL пользователя как удалённые.
func (r *DBRepository) BatchDeleteByUserID(userID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	query := `UPDATE short_urls SET is_deleted = TRUE WHERE id = ANY($1) AND user_id = $2`
	_, err := r.db.Exec(query, pq.Array(ids), userID)
	return err
}
