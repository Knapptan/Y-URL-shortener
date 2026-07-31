package config

import (
	"flag"
	"os"
)

const (
	DefaultServerAddress = "localhost:8080"
	DefaultBaseURL       = "http://localhost:8080"

	EnvServerAddress = "SERVER_ADDRESS"
	EnvBaseURL       = "BASE_URL"
)

type Config struct {
	ServerAddress string
	BaseURL       string
}

// BuildConfig определяет конфигурацию на основе значений из флагов и переменных окружения.
// Приоритет: переменная окружения > флаг > значение по умолчанию.
func BuildConfig(flagServerAddress, flagBaseURL string) *Config {
	return &Config{
		ServerAddress: DefaultServerAddress,
		BaseURL:       DefaultBaseURL,
	}
}

// resolveConfig определяет итоговые значения с учётом приоритета:
// переменная окружения > переданный флаг > значение по умолчанию.
func resolveConfig(flagServer, flagBase, envServer, envBase string) *Config {
	server := flagServer
	if envServer != "" {
		server = envServer
	}
	base := flagBase
	if envBase != "" {
		base = envBase
	}
	return &Config{
		ServerAddress: server,
		BaseURL:       base,
	}
}

// ParseFlags парсит флаги командной строки и переменные окружения,
// возвращает конфигурацию с учётом приоритета.
func ParseFlags() *Config {
	var (
		serverAddress string
		baseURL       string
	)

	flag.StringVar(&serverAddress, "a", DefaultServerAddress, "address and port")
	flag.StringVar(&baseURL, "b", DefaultBaseURL, "base URL")
	flag.Parse()

	envServer := os.Getenv(EnvServerAddress)
	envBase := os.Getenv(EnvBaseURL)

	return resolveConfig(serverAddress, baseURL, envServer, envBase)
}
