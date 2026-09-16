package app

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"github.com/Knapptan/Y-URL-shortener/internal/config"
	"github.com/Knapptan/Y-URL-shortener/internal/handler"
	"github.com/Knapptan/Y-URL-shortener/internal/middleware"
	"github.com/Knapptan/Y-URL-shortener/internal/repository"
	"github.com/Knapptan/Y-URL-shortener/internal/service"
)

// Run запускает HTTP-сервер с заданной конфигурацией.
func Run(cfg *config.Config) error {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	var repo repository.URLRepository
	var db *sql.DB

	if cfg.DatabaseDSN != "" {
		var err error
		db, err = sql.Open("postgres", cfg.DatabaseDSN)
		if err != nil {
			slog.Error("Failed to open DB", "error", err)
			return err
		}
		if err := db.Ping(); err != nil {
			slog.Error("Failed to ping DB", "error", err)
			return err
		}
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		m, err := migrate.New("file://"+filepath.Join(cwd, "migrations"), cfg.DatabaseDSN)
		if err != nil {
			slog.Error("Failed to init migrations", "error", err)
			return err
		}
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			slog.Error("Failed to apply migrations", "error", err)
			return err
		}
		dbRepo, err := repository.NewDBRepository(db)
		if err != nil {
			slog.Error("Failed to init DB repository", "error", err)
			return err
		}
		repo = dbRepo
		slog.Info("Using PostgreSQL storage")
	} else if cfg.FileStoragePath != "" {
		fileRepo, err := repository.NewFileRepository(cfg.FileStoragePath)
		if err != nil {
			slog.Error("Failed to init file storage", "error", err)
			return err
		}
		repo = fileRepo
		slog.Info("Using file storage", "path", cfg.FileStoragePath)
	} else {
		repo = repository.NewInMemoryRepo()
		slog.Info("Using in-memory storage")
	}

	defer func() {
		if db != nil {
			if err := db.Close(); err != nil {
				slog.Error("Failed to close DB", "error", err)
			}
		}
	}()

	svc := service.NewURLService(repo, cfg.BaseURL)
	h := handler.NewURLHandler(svc, cfg.BaseURL)
	pingHandler := handler.NewPingHandler(db)

	r := chi.NewRouter()
	r.Use(middleware.AuthMiddleware(cfg.SecretKey))
	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.GzipMiddleware)

	r.Get("/ping", pingHandler.Ping)
	r.Post("/", h.CreateShortURL)
	r.Get("/{id}", h.RedirectToOriginal)
	r.Post("/api/shorten", h.CreateShortenJSON)
	r.Post("/api/shorten/batch", h.CreateShortenBatch)
	r.Get("/api/user/urls", h.GetUserURLs)
	r.Delete("/api/user/urls", h.DeleteUserURLs)

	slog.Info("Starting server", "address", cfg.ServerAddress)
	return http.ListenAndServe(cfg.ServerAddress, r)
}
