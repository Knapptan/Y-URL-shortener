package app

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"

	_ "github.com/lib/pq"

	"github.com/go-chi/chi/v5"

	"github.com/Knapptan/Y-URL-shortener/internal/config"
	"github.com/Knapptan/Y-URL-shortener/internal/handler"
	"github.com/Knapptan/Y-URL-shortener/internal/middleware"
	"github.com/Knapptan/Y-URL-shortener/internal/repository"
	"github.com/Knapptan/Y-URL-shortener/internal/service"
)

// Run запускает HTTP-сервер с заданной конфигурацией.
func Run(cfg *config.Config) error {
	// Настройка логгера
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	var repo repository.URLRepository
	var db *sql.DB

	// 1. Если задан DSN – используем БД
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
		dbRepo, err := repository.NewDBRepository(db)
		if err != nil {
			slog.Error("Failed to init DB repository", "error", err)
			return err
		}
		repo = dbRepo
		slog.Info("Using PostgreSQL storage")
	} else if cfg.FileStoragePath != "" {
		// 2. Иначе если есть путь к файлу – используем файл
		fileRepo, err := repository.NewFileRepository(cfg.FileStoragePath)
		if err != nil {
			slog.Error("Failed to init file storage", "error", err)
			return err
		}
		repo = fileRepo
		slog.Info("Using file storage", "path", cfg.FileStoragePath)
	} else {
		// 3. Иначе – in‑memory
		repo = repository.NewInMemoryRepo()
		slog.Info("Using in‑memory storage")
	}

	// Закрываем БД при завершении (если она была открыта)
	defer func() {
		if db != nil {
			if err := db.Close(); err != nil {
				slog.Error("Failed to close DB", "error", err)
			}
		}
	}()

	// Инициализация сервиса и хендлеров (как раньше)
	svc := service.NewURLService(repo)
	h := handler.NewURLHandler(svc, cfg.BaseURL)
	pingHandler := handler.NewPingHandler(db)

	r := chi.NewRouter()
	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.GzipMiddleware)

	r.Get("/ping", pingHandler.Ping)
	r.Post("/", h.CreateShortURL)
	r.Get("/{id}", h.RedirectToOriginal)
	r.Post("/api/shorten", h.CreateShortenJSON)

	slog.Info("Starting server", "address", cfg.ServerAddress)
	return http.ListenAndServe(cfg.ServerAddress, r)
}
