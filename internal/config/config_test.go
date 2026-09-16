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
		flagSecret     string
		envSecret      string
		expectedServer string
		expectedBase   string
		expectedFile   string
		expectedDSN    string
		expectedSecret string
	}{
		{
			name:           "no flags, no env",
			flagServer:     DefaultServerAddress,
			flagBase:       DefaultBaseURL,
			flagFile:       "",
			flagDSN:        "",
			flagSecret:     DefaultSecretKey,
			expectedServer: DefaultServerAddress,
			expectedBase:   DefaultBaseURL,
			expectedFile:   DefaultFileStoragePath,
			expectedDSN:    "",
			expectedSecret: DefaultSecretKey,
		},
		{
			name:           "flag only",
			flagServer:     "localhost:8888",
			flagBase:       "http://localhost:8888",
			flagFile:       "",
			flagDSN:        "postgres://user:pass@localhost/db",
			flagSecret:     "flag-secret",
			expectedServer: "localhost:8888",
			expectedBase:   "http://localhost:8888",
			expectedFile:   DefaultFileStoragePath,
			expectedDSN:    "postgres://user:pass@localhost/db",
			expectedSecret: "flag-secret",
		},
		{
			name:           "env wins",
			flagServer:     "localhost:8888",
			flagBase:       "http://localhost:8888",
			flagSecret:     "flag-secret",
			envServer:      "localhost:9999",
			envBase:        "http://localhost:9999",
			envSecret:      "env-secret",
			expectedServer: "localhost:9999",
			expectedBase:   "http://localhost:9999",
			expectedFile:   DefaultFileStoragePath,
			expectedSecret: "env-secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := resolveConfig(
				tt.flagServer, tt.flagBase, tt.envServer, tt.envBase,
				tt.flagFile, tt.envFile, tt.flagDSN, tt.envDSN,
				tt.flagSecret, tt.envSecret,
			)
			assert.Equal(t, tt.expectedServer, cfg.ServerAddress)
			assert.Equal(t, tt.expectedBase, cfg.BaseURL)
			assert.Equal(t, tt.expectedFile, cfg.FileStoragePath)
			assert.Equal(t, tt.expectedDSN, cfg.DatabaseDSN)
			assert.Equal(t, tt.expectedSecret, cfg.SecretKey)
		})
	}
}

func TestParseFlags(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	t.Run("flag overrides default", func(t *testing.T) {
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"cmd", "-a", "localhost:8888", "-b", "http://localhost:8888", "-s", "my-secret"}
		t.Setenv(EnvServerAddress, "")
		t.Setenv(EnvBaseURL, "")
		t.Setenv(EnvFileStoragePath, "")
		t.Setenv(EnvDatabaseDSN, "")
		t.Setenv(EnvSecretKey, "")
		cfg := ParseFlags()
		assert.Equal(t, "localhost:8888", cfg.ServerAddress)
		assert.Equal(t, "http://localhost:8888", cfg.BaseURL)
		assert.Equal(t, DefaultFileStoragePath, cfg.FileStoragePath)
		assert.Equal(t, "my-secret", cfg.SecretKey)
	})

	t.Run("env overrides flag", func(t *testing.T) {
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"cmd", "-a", "localhost:8888", "-b", "http://localhost:8888", "-s", "flag-secret"}
		t.Setenv(EnvServerAddress, "localhost:9999")
		t.Setenv(EnvBaseURL, "http://localhost:9999")
		t.Setenv(EnvFileStoragePath, "")
		t.Setenv(EnvDatabaseDSN, "")
		t.Setenv(EnvSecretKey, "env-secret")
		cfg := ParseFlags()
		assert.Equal(t, "localhost:9999", cfg.ServerAddress)
		assert.Equal(t, "http://localhost:9999", cfg.BaseURL)
		assert.Equal(t, DefaultFileStoragePath, cfg.FileStoragePath)
		assert.Equal(t, "env-secret", cfg.SecretKey)
	})
}
