package config

import "flag"

const (
	// DefaultServerAddress значение по умолчанию для адреса сервера.
	DefaultServerAddress = "localhost:8080"
	// DefaultBaseURL значение по умолчанию для базового URL.
	DefaultBaseURL = "http://localhost:8080"
)

// Config хранит настройки сервера.
type Config struct {
	ServerAddress string // адрес для запуска сервера
	BaseURL       string // базовый URL для коротких ссылок
}

// ParseFlags обрабатывает флаги командной строки и возвращает конфигурацию.
func ParseFlags() *Config {
	var cfg Config

	// Регистрируем флаги с значениями по умолчанию
	flag.StringVar(&cfg.ServerAddress, "a", DefaultServerAddress, "address and port to run server")
	flag.StringVar(&cfg.BaseURL, "b", DefaultBaseURL, "base address for shortened URLs")

	flag.Parse()

	return &cfg
}
