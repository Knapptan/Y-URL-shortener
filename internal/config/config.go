// Package config обеспечивает парсинг конфигурации сервиса через флаги CLI
// и переменные окружения.
package config

import (
	"flag"
	"os"
)

// Константы конфигурации: значения по умолчанию и имена переменных окружения.
const (
	// DefaultServerAddress — адрес сервера по умолчанию.
	DefaultServerAddress = "localhost:8080"
	// DefaultBaseURL — базовый URL по умолчанию.
	DefaultBaseURL = "http://localhost:8080"

	// EnvServerAddress — имя переменной окружения для адреса сервера.
	EnvServerAddress = "SERVER_ADDRESS"
	// EnvBaseURL — имя переменной окружения для базового URL.
	EnvBaseURL = "BASE_URL"

	// EnvFileStoragePath — имя переменной окружения для пути к файлу хранилища.
	EnvFileStoragePath = "FILE_STORAGE_PATH"
	// DefaultFileStoragePath — путь к файлу хранилища по умолчанию.
	DefaultFileStoragePath = "storage.json"

	// EnvDatabaseDSN — имя переменной окружения для DSN базы данных.
	EnvDatabaseDSN = "DATABASE_DSN"
	// DefaultDatabaseDSN — DSN базы данных по умолчанию (пустое значение отключает БД).
	DefaultDatabaseDSN = ""

	// EnvSecretKey — имя переменной окружения для секретного ключа подписи cookie.
	EnvSecretKey = "SECRET_KEY"
	// DefaultSecretKey — секретный ключ по умолчанию (только для локальной разработки).
	DefaultSecretKey = "your-secret-key"

	// EnvAuditFile — имя переменной окружения для пути к файлу аудита.
	EnvAuditFile = "AUDIT_FILE"
	// EnvAuditURL — имя переменной окружения для URL удалённого приёмника аудита.
	EnvAuditURL = "AUDIT_URL"

	// EnvEnableHTTPS — имя переменной окружения для включения HTTPS.
	EnvEnableHTTPS = "ENABLE_HTTPS"
)

// Config хранит настройки сервиса.
type Config struct {
	ServerAddress   string
	BaseURL         string
	FileStoragePath string
	DatabaseDSN     string
	SecretKey       string
	AuditFile       string
	AuditURL        string
	EnableHTTPS     bool
}

// resolveConfig собирает итоговую конфигурацию по приоритету:
// env > флаг > дефолт.
func resolveConfig(
	flagServer, flagBase, envServer, envBase string,
	flagFile, envFile, flagDSN, envDSN string,
	flagSecret, envSecret string,
	flagAuditFile, envAuditFile, flagAuditURL, envAuditURL string,
	flagEnableHTTPS, envEnableHTTPS bool,
) *Config {
	server := flagServer
	if envServer != "" {
		server = envServer
	}
	base := flagBase
	if envBase != "" {
		base = envBase
	}
	filePath := flagFile
	if envFile != "" {
		filePath = envFile
	}
	if filePath == "" {
		filePath = DefaultFileStoragePath
	}
	dsn := flagDSN
	if envDSN != "" {
		dsn = envDSN
	}
	secret := flagSecret
	if envSecret != "" {
		secret = envSecret
	}
	if secret == "" {
		secret = DefaultSecretKey
	}
	auditFile := flagAuditFile
	if envAuditFile != "" {
		auditFile = envAuditFile
	}
	auditURL := flagAuditURL
	if envAuditURL != "" {
		auditURL = envAuditURL
	}

	// HTTPS включается, если задан флаг -s или ENABLE_HTTPS=true.
	enableHTTPS := flagEnableHTTPS || envEnableHTTPS

	return &Config{
		ServerAddress:   server,
		BaseURL:         base,
		FileStoragePath: filePath,
		DatabaseDSN:     dsn,
		SecretKey:       secret,
		AuditFile:       auditFile,
		AuditURL:        auditURL,
		EnableHTTPS:     enableHTTPS,
	}
}

// ParseFlags парсит флаги CLI и переменные окружения.
func ParseFlags() *Config {
	var (
		serverAddress string
		baseURL       string
		filePath      string
		databaseDSN   string
		secretKey     string
		auditFile     string
		auditURL      string
		enableHTTPS   bool
	)
	flag.StringVar(&serverAddress, "a", DefaultServerAddress, "address and port")
	flag.StringVar(&baseURL, "b", DefaultBaseURL, "base URL")
	flag.StringVar(&filePath, "f", DefaultFileStoragePath, "file path for storage")
	flag.StringVar(&databaseDSN, "d", DefaultDatabaseDSN, "database DSN")
	// Secret key — переименован с -s на -k, чтобы освободить -s под HTTPS.
	flag.StringVar(&secretKey, "k", DefaultSecretKey, "secret key for signing cookies")
	flag.StringVar(&auditFile, "audit-file", "", "audit log file path")
	flag.StringVar(&auditURL, "audit-url", "", "audit remote server URL")
	// -s — включение HTTPS (по ТЗ инкремента).
	flag.BoolVar(&enableHTTPS, "s", false, "enable HTTPS")
	flag.Parse()

	envServer, _ := os.LookupEnv(EnvServerAddress)
	envBase, _ := os.LookupEnv(EnvBaseURL)
	envFile, _ := os.LookupEnv(EnvFileStoragePath)
	envDSN, _ := os.LookupEnv(EnvDatabaseDSN)
	envSecret, _ := os.LookupEnv(EnvSecretKey)
	envAuditFile, _ := os.LookupEnv(EnvAuditFile)
	envAuditURL, _ := os.LookupEnv(EnvAuditURL)
	envHTTPS := os.Getenv(EnvEnableHTTPS) == "true" || os.Getenv(EnvEnableHTTPS) == "1"

	return resolveConfig(
		serverAddress, baseURL, envServer, envBase,
		filePath, envFile, databaseDSN, envDSN,
		secretKey, envSecret,
		auditFile, envAuditFile, auditURL, envAuditURL,
		enableHTTPS, envHTTPS,
	)
}
