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
		expectedServer string
		expectedBase   string
	}{
		{
			name:           "no flags, no env",
			flagServer:     DefaultServerAddress,
			flagBase:       DefaultBaseURL,
			envServer:      "",
			envBase:        "",
			expectedServer: DefaultServerAddress,
			expectedBase:   DefaultBaseURL,
		},
		{
			name:           "flag only",
			flagServer:     "localhost:8888",
			flagBase:       "http://localhost:8888",
			envServer:      "",
			envBase:        "",
			expectedServer: "localhost:8888",
			expectedBase:   "http://localhost:8888",
		},
		{
			name:           "env only",
			flagServer:     DefaultServerAddress,
			flagBase:       DefaultBaseURL,
			envServer:      "localhost:9999",
			envBase:        "http://localhost:9999",
			expectedServer: "localhost:9999",
			expectedBase:   "http://localhost:9999",
		},
		{
			name:           "flag and env – env wins",
			flagServer:     "localhost:8888",
			flagBase:       "http://localhost:8888",
			envServer:      "localhost:9999",
			envBase:        "http://localhost:9999",
			expectedServer: "localhost:9999",
			expectedBase:   "http://localhost:9999",
		},
		{
			name:           "env only for server, flag only for base",
			flagServer:     "localhost:8888",
			flagBase:       "http://localhost:8888",
			envServer:      "localhost:7777",
			envBase:        "",
			expectedServer: "localhost:7777",
			expectedBase:   "http://localhost:8888",
		},
		{
			name:           "empty env values ignored",
			flagServer:     "localhost:8888",
			flagBase:       "http://localhost:8888",
			envServer:      "",
			envBase:        "http://localhost:9999",
			expectedServer: "localhost:8888",
			expectedBase:   "http://localhost:9999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := resolveConfig(tt.flagServer, tt.flagBase, tt.envServer, tt.envBase)
			assert.Equal(t, tt.expectedServer, cfg.ServerAddress)
			assert.Equal(t, tt.expectedBase, cfg.BaseURL)
		})
	}
}

func TestParseFlags(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	t.Run("flag overrides default", func(t *testing.T) {
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"cmd", "-a", "localhost:8888", "-b", "http://localhost:8888"}
		t.Setenv(EnvServerAddress, "")
		t.Setenv(EnvBaseURL, "")
		cfg := ParseFlags()
		assert.Equal(t, "localhost:8888", cfg.ServerAddress)
		assert.Equal(t, "http://localhost:8888", cfg.BaseURL)
	})

	t.Run("env overrides flag", func(t *testing.T) {
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"cmd", "-a", "localhost:8888", "-b", "http://localhost:8888"}
		t.Setenv(EnvServerAddress, "localhost:9999")
		t.Setenv(EnvBaseURL, "http://localhost:9999")
		cfg := ParseFlags()
		assert.Equal(t, "localhost:9999", cfg.ServerAddress)
		assert.Equal(t, "http://localhost:9999", cfg.BaseURL)
	})
}
