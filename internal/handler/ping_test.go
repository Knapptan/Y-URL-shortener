package handler

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pingDriver struct{}

func (pingDriver) Open(_ string) (driver.Conn, error) {
	return pingConn{}, nil
}

type pingConn struct{}

func (pingConn) Prepare(_ string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}

func (pingConn) Close() error {
	return nil
}

func (pingConn) Begin() (driver.Tx, error) {
	return nil, errors.New("not implemented")
}

func (pingConn) Ping(_ context.Context) error {
	return nil
}

func newFakeDB(t *testing.T) *sql.DB {
	t.Helper()

	name := fmt.Sprintf("fake-ping-%d", time.Now().UnixNano())
	sql.Register(name, pingDriver{})
	db, err := sql.Open(name, "")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func TestPingHandler(t *testing.T) {
	t.Run("database is not configured", func(t *testing.T) {
		handler := NewPingHandler(nil)
		recorder := httptest.NewRecorder()

		handler.Ping(recorder, httptest.NewRequest(http.MethodGet, "/ping", nil))

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Equal(t, "База данных не настроена\n", recorder.Body.String())
	})

	t.Run("database is unavailable", func(t *testing.T) {
		db := newFakeDB(t)
		require.NoError(t, db.Close())

		handler := NewPingHandler(db)
		recorder := httptest.NewRecorder()

		handler.Ping(recorder, httptest.NewRequest(http.MethodGet, "/ping", nil))

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Equal(t, "База данных недоступна\n", recorder.Body.String())
	})

	t.Run("database is available", func(t *testing.T) {
		handler := NewPingHandler(newFakeDB(t))
		recorder := httptest.NewRecorder()

		handler.Ping(recorder, httptest.NewRequest(http.MethodGet, "/ping", nil))

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "OK", recorder.Body.String())
	})
}
