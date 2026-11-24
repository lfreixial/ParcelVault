package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func cleanEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"CONFIG_FILE", "LISTEN_ADDR", "LOG_LEVEL", "SHUTDOWN_TIMEOUT"} {
		name := "PARCELVAULT_" + key
		t.Setenv(name, "")
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
	}
}

func configFile(t *testing.T, content string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PARCELVAULT_CONFIG_FILE", path)
}

func TestLoadPrecedence(t *testing.T) {
	cleanEnv(t)
	cfg, err := Load()
	if err != nil || cfg != (Config{"127.0.0.1:8080", slog.LevelInfo, 10 * time.Second}) {
		t.Fatalf("defaults: %+v, %v", cfg, err)
	}
	configFile(t, `{"listen_addr":"[::1]:9090","log_level":"debug","shutdown_timeout":"2s"}`)
	t.Setenv("PARCELVAULT_LOG_LEVEL", "WARN")
	cfg, err = Load()
	if err != nil || cfg != (Config{"[::1]:9090", slog.LevelWarn, 2 * time.Second}) {
		t.Fatalf("overrides: %+v, %v", cfg, err)
	}
}

func TestInvalidConfig(t *testing.T) {
	cleanEnv(t)
	for _, content := range []string{
		`null`, `[]`, `{"private-marker": "private-marker"}`, `{}` + `{}`,
		`{"log_level":null}`, `{"log_level":2}`, `{"log_level":"private-marker"}`,
		`{"listen_addr":"private-marker"}`, `{"listen_addr":":80"}`,
		`{"listen_addr":"127.0.0.1:0"}`, `{"listen_addr":"127.0.0.1:65536"}`,
		`{"shutdown_timeout":"0s"}`, `{"shutdown_timeout":"31s"}`,
		`{"shutdown_timeout":"private-marker"}`, strings.Repeat(" ", maxFileSize) + "{}",
	} {
		configFile(t, content)
		assertInvalid(t)
	}
	t.Setenv("PARCELVAULT_CONFIG_FILE", filepath.Join(t.TempDir(), "private-marker"))
	assertInvalid(t)
}

func TestEmptyEnvOverride(t *testing.T) {
	cleanEnv(t)
	t.Setenv("PARCELVAULT_LOG_LEVEL", "")
	assertInvalid(t)
}

func assertInvalid(t *testing.T) {
	t.Helper()
	if _, err := Load(); err == nil || strings.Contains(err.Error(), "private-marker") {
		t.Fatalf("expected a sanitized configuration error, got %v", err)
	}
}
