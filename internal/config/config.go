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
func resolveConfig(
	flagServer, flagBase, envServer, envBase,
	flagFile, envFile, flagDSN, envDSN,
	flagSecret, envSecret, flagAuditFile, envAuditFile,
	flagAuditURL, envAuditURL string,
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
	return &Config{
		ServerAddress:   server,
		BaseURL:         base,
		FileStoragePath: filePath,
		DatabaseDSN:     dsn,
		SecretKey:       secret,
		AuditFile:       auditFile,
		AuditURL:        auditURL,
	}
}

// ParseFlags парсит флаги командной строки и переменные окружения,
// возвращает конфигурацию с учётом приоритета.
func ParseFlags() *Config {
	var (
		serverAddress string
		baseURL       string
		filePath      string
		databaseDSN   string
		secretKey     string
		auditFile     string
		auditURL      string
	)
	flag.StringVar(&serverAddress, "a", DefaultServerAddress, "address and port")
	flag.StringVar(&baseURL, "b", DefaultBaseURL, "base URL")
	flag.StringVar(&filePath, "f", DefaultFileStoragePath, "file path for storage")
	flag.StringVar(&databaseDSN, "d", DefaultDatabaseDSN, "database DSN")
	flag.StringVar(&secretKey, "s", DefaultSecretKey, "secret key for signing cookies")
	flag.StringVar(&auditFile, "audit-file", "", "audit log file path")
	flag.StringVar(&auditURL, "audit-url", "", "audit remote server URL")
	flag.Parse()

	envServer, _ := os.LookupEnv(EnvServerAddress)
	envBase, _ := os.LookupEnv(EnvBaseURL)
	envFile, _ := os.LookupEnv(EnvFileStoragePath)
	envDSN, _ := os.LookupEnv(EnvDatabaseDSN)
	envSecret, _ := os.LookupEnv(EnvSecretKey)
	envAuditFile, _ := os.LookupEnv(EnvAuditFile)
	envAuditURL, _ := os.LookupEnv(EnvAuditURL)

	return resolveConfig(
		serverAddress, baseURL, envServer, envBase,
		filePath, envFile, databaseDSN, envDSN,
		secretKey, envSecret, auditFile, envAuditFile,
		auditURL, envAuditURL,
	)
}
