// Package app обеспечивает инициализацию приложения: загрузку конфигурации,
// настройку репозиториев, построение роутера chi и запуск HTTP/HTTPS-сервера
// с корректным graceful shutdown по сигналам.
package app

import (
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"github.com/Knapptan/Y-URL-shortener/internal/audit"
	"github.com/Knapptan/Y-URL-shortener/internal/cert"
	"github.com/Knapptan/Y-URL-shortener/internal/config"
	"github.com/Knapptan/Y-URL-shortener/internal/handler"
	"github.com/Knapptan/Y-URL-shortener/internal/middleware"
	"github.com/Knapptan/Y-URL-shortener/internal/repository"
	"github.com/Knapptan/Y-URL-shortener/internal/service"
)

// shutdownTimeout — максимальное время ожидания завершения активных запросов.
const shutdownTimeout = 30 * time.Second

// Run запускает HTTP/HTTPS-сервер с graceful shutdown по сигналам
// SIGTERM, SIGINT, SIGQUIT.
func Run(cfg *config.Config) error {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	// === Репозиторий ===
	var repo repository.URLRepository
	var db *sql.DB

	if cfg.DatabaseDSN != "" {
		var err error
		db, err = sql.Open("postgres", cfg.DatabaseDSN)
		if err != nil {
			return fmt.Errorf("open DB: %w", err)
		}
		if err := db.Ping(); err != nil {
			return fmt.Errorf("ping DB: %w", err)
		}
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		m, err := migrate.New("file://"+filepath.Join(cwd, "migrations"), cfg.DatabaseDSN)
		if err != nil {
			return fmt.Errorf("init migrations: %w", err)
		}
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("apply migrations: %w", err)
		}
		dbRepo, err := repository.NewDBRepository(db)
		if err != nil {
			return fmt.Errorf("init DB repository: %w", err)
		}
		repo = dbRepo
		slog.Info("Using PostgreSQL storage")
	} else if cfg.FileStoragePath != "" {
		fileRepo, err := repository.NewFileRepository(cfg.FileStoragePath)
		if err != nil {
			return fmt.Errorf("init file storage: %w", err)
		}
		repo = fileRepo
		slog.Info("Using file storage", "path", cfg.FileStoragePath)
	} else {
		repo = repository.NewInMemoryRepo()
		slog.Info("Using in-memory storage")
	}

	// Регистрируем деферы в обратном порядке приоритета закрытия:
	// сначала наблюдатели аудита, затем БД.
	if db != nil {
		defer func() {
			if err := db.Close(); err != nil {
				slog.Error("close DB", "error", err)
			}
		}()
	}

	// === Аудит ===
	publisher := audit.NewPublisher()
	if cfg.AuditFile != "" {
		fileObserver, err := audit.NewFileObserver(cfg.AuditFile)
		if err != nil {
			return fmt.Errorf("init audit file: %w", err)
		}
		defer func() {
			if err := fileObserver.Close(); err != nil {
				slog.Error("close audit file", "error", err)
			}
		}()
		publisher.Register(fileObserver)
		slog.Info("audit: file enabled", "path", cfg.AuditFile)
	}
	if cfg.AuditURL != "" {
		httpObserver := audit.NewHTTPObserver(cfg.AuditURL)
		defer func() {
			if err := httpObserver.Close(); err != nil {
				slog.Error("close audit http", "error", err)
			}
		}()
		publisher.Register(httpObserver)
		slog.Info("audit: http enabled", "url", cfg.AuditURL)
	}

	// === Сервис и хендлеры ===
	svc := service.NewURLService(repo, cfg.BaseURL)
	h := handler.NewURLHandler(svc, cfg.BaseURL, publisher)
	pingHandler := handler.NewPingHandler(db)

	// Обработчик внутренней статистики (только для доверенной подсети).
	statsHandler, err := handler.NewInternalStatsHandler(svc, cfg.TrustedSubnet)
	if err != nil {
		slog.Error("Failed to init stats handler", "error", err, "subnet", cfg.TrustedSubnet)
		return err
	}

	// === Роутер ===
	r := chi.NewRouter()
	r.Use(middleware.AuthMiddleware(cfg.SecretKey))
	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.GzipMiddleware)

	r.Get("/ping", pingHandler.Ping)
	r.Post("/", h.CreateShortURL)
	r.Post("/api/shorten", h.CreateShortenJSON)
	r.Post("/api/shorten/batch", h.CreateShortenBatch)
	r.Post("/api/shorten/async", h.AsyncShorten)
	r.Get("/api/shorten/result/{correlation_id}", h.GetAsyncResult)
	r.Get("/api/user/urls", h.GetUserURLs)
	r.Delete("/api/user/urls", h.DeleteUserURLs)
	// Статистика — после конкретных /api/..., но до catch-all /{id}.
	r.Method(http.MethodGet, "/api/internal/stats", statsHandler)
	r.Get("/{id}", h.RedirectToOriginal)

	// === HTTP(S)-сервер ===
	srv := &http.Server{
		Addr:              cfg.ServerAddress,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Контекст, отменяемый по SIGTERM/SIGINT/SIGQUIT.
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer stop()

	// Запускаем сервер в фоновой горутине.
	errCh := make(chan error, 1)
	go func() {
		errCh <- serve(srv, cfg.EnableHTTPS)
	}()

	// Ждём сигнал или фатальную ошибку сервера.
	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received, draining connections...")
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server: %w", err)
		}
		return nil
	}

	// Graceful shutdown: перестаём принимать новые соединения,
	// ждём завершения активных запросов (до shutdownTimeout).
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed, forcing close", "error", err)
		_ = srv.Close()
	}

	slog.Info("shutdown complete")

	// Деферы выполнятся при выходе из Run:
	// fileObserver.Close() — дождётся записи всех событий в файл;
	// httpObserver.Close() — дождётся отправки всех событий;
	// db.Close() — закроет соединения с БД.
	return nil
}

// serve запускает сервер в HTTP или HTTPS режиме.
func serve(srv *http.Server, useTLS bool) error {
	if useTLS {
		c, err := cert.GenerateSelfSigned()
		if err != nil {
			return fmt.Errorf("generate cert: %w", err)
		}
		srv.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{c},
			MinVersion:   tls.VersionTLS12,
		}
		ln, err := tls.Listen("tcp", srv.Addr, srv.TLSConfig)
		if err != nil {
			return fmt.Errorf("tls listen: %w", err)
		}
		slog.Info("starting HTTPS server", "addr", srv.Addr)
		return srv.Serve(ln)
	}
	slog.Info("starting HTTP server", "addr", srv.Addr)
	return srv.ListenAndServe()
}
