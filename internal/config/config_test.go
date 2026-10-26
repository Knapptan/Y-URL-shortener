package config

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveConfig(t *testing.T) {
	tests := []struct {
		name           string
		flagServer     string
		flagBase       string
		envServer      string
		envBase        string
		flagFile       string
		envFile        string
		flagDSN        string
		envDSN         string
		expectedServer string
		expectedBase   string
		expectedFile   string
		expectedDSN    string
	}{
		{
			name:           "no flags, no env",
			flagServer:     DefaultServerAddress,
			flagBase:       DefaultBaseURL,
			envServer:      "",
			envBase:        "",
			flagFile:       "",
			envFile:        "",
			flagDSN:        "",
			envDSN:         "",
			expectedServer: DefaultServerAddress,
			expectedBase:   DefaultBaseURL,
			expectedFile:   DefaultFileStoragePath,
			expectedDSN:    "",
		},
		{
			name:           "flag only",
			flagServer:     "localhost:8888",
			flagBase:       "http://localhost:8888",
			envServer:      "",
			envBase:        "",
			flagFile:       "",
			envFile:        "",
			flagDSN:        "postgres://user:pass@localhost/db",
			envDSN:         "",
			expectedServer: "localhost:8888",
			expectedBase:   "http://localhost:8888",
			expectedFile:   DefaultFileStoragePath,
			expectedDSN:    "postgres://user:pass@localhost/db",
		},
		{
			name:           "env only",
			flagServer:     DefaultServerAddress,
			flagBase:       DefaultBaseURL,
			envServer:      "localhost:9999",
			envBase:        "http://localhost:9999",
			flagFile:       "",
			envFile:        "",
			flagDSN:        "",
			envDSN:         "postgres://user:pass@env/db",
			expectedServer: "localhost:9999",
			expectedBase:   "http://localhost:9999",
			expectedFile:   DefaultFileStoragePath,
			expectedDSN:    "postgres://user:pass@env/db",
		},
		{
			name:           "flag and env – env wins for DSN",
			flagServer:     "localhost:8888",
			flagBase:       "http://localhost:8888",
			envServer:      "localhost:9999",
			envBase:        "http://localhost:9999",
			flagFile:       "",
			envFile:        "",
			flagDSN:        "postgres://flag/db",
			envDSN:         "postgres://env/db",
			expectedServer: "localhost:9999",
			expectedBase:   "http://localhost:9999",
			expectedFile:   DefaultFileStoragePath,
			expectedDSN:    "postgres://env/db",
		},
		{
			name:           "DSN flag only, no env",
			flagServer:     DefaultServerAddress,
			flagBase:       DefaultBaseURL,
			envServer:      "",
			envBase:        "",
			flagFile:       "",
			envFile:        "",
			flagDSN:        "postgres://flagonly/db",
			envDSN:         "",
			expectedServer: DefaultServerAddress,
			expectedBase:   DefaultBaseURL,
			expectedFile:   DefaultFileStoragePath,
			expectedDSN:    "postgres://flagonly/db",
		},
		{
			name:           "DSN env only, no flag",
			flagServer:     DefaultServerAddress,
			flagBase:       DefaultBaseURL,
			envServer:      "",
			envBase:        "",
			flagFile:       "",
			envFile:        "",
			flagDSN:        "",
			envDSN:         "postgres://envonly/db",
			expectedServer: DefaultServerAddress,
			expectedBase:   DefaultBaseURL,
			expectedFile:   DefaultFileStoragePath,
			expectedDSN:    "postgres://envonly/db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := resolveConfig(tt.flagServer, tt.flagBase, tt.envServer, tt.envBase,
				tt.flagFile, tt.envFile, tt.flagDSN, tt.envDSN)
			assert.Equal(t, tt.expectedServer, cfg.ServerAddress)
			assert.Equal(t, tt.expectedBase, cfg.BaseURL)
			assert.Equal(t, tt.expectedFile, cfg.FileStoragePath)
			assert.Equal(t, tt.expectedDSN, cfg.DatabaseDSN)
		})
	}
}

func TestParseFlags(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	t.Run("flag overrides default", func(t *testing.T) {
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"cmd", "-a", "localhost:8888", "-b", "http://localhost:8888", "-d", "postgres://flag/db"}
		t.Setenv(EnvServerAddress, "")
		t.Setenv(EnvBaseURL, "")
		t.Setenv(EnvFileStoragePath, "")
		t.Setenv(EnvDatabaseDSN, "")
		cfg := ParseFlags()
		assert.Equal(t, "localhost:8888", cfg.ServerAddress)
		assert.Equal(t, "http://localhost:8888", cfg.BaseURL)
		assert.Equal(t, DefaultFileStoragePath, cfg.FileStoragePath)
		assert.Equal(t, "postgres://flag/db", cfg.DatabaseDSN)
	})

	t.Run("env overrides flag", func(t *testing.T) {
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"cmd", "-a", "localhost:8888", "-b", "http://localhost:8888", "-d", "postgres://flag/db"}
		t.Setenv(EnvServerAddress, "localhost:9999")
		t.Setenv(EnvBaseURL, "http://localhost:9999")
		t.Setenv(EnvFileStoragePath, "")
		t.Setenv(EnvDatabaseDSN, "postgres://env/db")
		cfg := ParseFlags()
		assert.Equal(t, "localhost:9999", cfg.ServerAddress)
		assert.Equal(t, "http://localhost:9999", cfg.BaseURL)
		assert.Equal(t, DefaultFileStoragePath, cfg.FileStoragePath)
		assert.Equal(t, "postgres://env/db", cfg.DatabaseDSN)
	})
}
