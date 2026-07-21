package config

import "flag"

// Config хранит настройки сервера
type Config struct {
	ServerAddress string // адрес для запуска сервера
	BaseURL       string // базовый URL для коротких ссылок
}

// ParseFlags обрабатывает флаги командной строки и возвращает конфигурацию.
func ParseFlags() *Config {
	var cfg Config

	// Регистрируем флаги с значениями по умолчанию
	flag.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "address and port to run server")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080", "base address for shortened URLs")

	flag.Parse()

	return &cfg
}
