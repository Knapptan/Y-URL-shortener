package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPick проверяет ключевую логику приоритетов для строк.
// Особое внимание — кейсу "флаг явно передан, но равен дефолту":
// файл НЕ должен побеждать флаг.
func TestPick(t *testing.T) {
	cases := []struct {
		name     string
		env      string
		flagVal  string
		flagSet  bool
		fileVal  string
		def      string
		expected string
	}{
		{"env beats all", "env", "flag", true, "file", "def", "env"},
		{"explicit flag beats file", "", "flag", true, "file", "def", "flag"},
		{"explicit flag==default beats file", "", "def", true, "file", "def", "def"},
		{"file beats default", "", "def", false, "file", "def", "file"},
		{"default only", "", "def", false, "", "def", "def"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, pick(c.env, c.flagVal, c.flagSet, c.fileVal, c.def))
		})
	}
}

// TestPickBool проверяет приоритеты для булевых значений.
func TestPickBool(t *testing.T) {
	cases := []struct {
		name     string
		envVal   bool
		envSet   bool
		flagVal  bool
		flagSet  bool
		fileVal  bool
		expected bool
	}{
		{"env=true wins", true, true, false, true, false, true},
		{"env explicitly false wins over flag", false, true, true, true, true, false},
		{"flag true when env not set", false, false, true, true, false, true},
		{"file when neither env nor flag", false, false, false, false, true, true},
		{"all false", false, false, false, false, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, pickBool(c.envVal, c.envSet, c.flagVal, c.flagSet, c.fileVal))
		})
	}
}

// TestLoadFileConfig проверяет чтение JSON-конфига.
func TestLoadFileConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
		"server_address": "file:9090",
		"base_url": "http://file:9090",
		"file_storage_path": "./file.json",
		"database_dsn": "postgres://file/db",
		"secret_key": "file-secret",
		"audit_file": "/tmp/file.log",
		"audit_url": "http://audit:8080",
		"enable_https": true
	}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	fc, err := loadFileConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "file:9090", fc.ServerAddress)
	assert.Equal(t, "http://file:9090", fc.BaseURL)
	assert.Equal(t, "./file.json", fc.FileStoragePath)
	assert.Equal(t, "postgres://file/db", fc.DatabaseDSN)
	assert.Equal(t, "file-secret", fc.SecretKey)
	assert.Equal(t, "/tmp/file.log", fc.AuditFile)
	assert.Equal(t, "http://audit:8080", fc.AuditURL)
	assert.True(t, fc.EnableHTTPS)
}

// TestLoadFileConfig_NotFound — отсутствующий файл возвращает ошибку.
func TestLoadFileConfig_NotFound(t *testing.T) {
	_, err := loadFileConfig(filepath.Join(t.TempDir(), "missing.json"))
	assert.Error(t, err)
}
