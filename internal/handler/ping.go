package handler

import (
	"database/sql"
	"net/http"
)

type PingHandler struct {
	db *sql.DB
}

func NewPingHandler(db *sql.DB) *PingHandler {
	return &PingHandler{db: db}
}

func (ph *PingHandler) Ping(w http.ResponseWriter, r *http.Request) {
	if ph.db == nil {
		http.Error(w, "База данных не настроена", http.StatusInternalServerError)
		return
	}
	if err := ph.db.PingContext(r.Context()); err != nil {
		http.Error(w, "База данных недоступна", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
