// Package config loads and validates the startup configuration before the server binds.
package config

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/netip"
	"os"
	"strings"
	"time"
)

type Config struct {
	ListenAddr      string
	LogLevel        slog.Level
	ShutdownTimeout time.Duration
}

const maxFileSize = 64 << 10

func Load() (Config, error) {
	values := map[string]string{
		"listen_addr":      "127.0.0.1:8080",
		"log_level":        "info",
		"shutdown_timeout": "10s",
	}
	if path, ok := os.LookupEnv("PARCELVAULT_CONFIG_FILE"); ok {
		if err := readFile(path, values); err != nil {
			return Config{}, err
		}
	}
	for key := range values {
		if value, ok := os.LookupEnv("PARCELVAULT_" + strings.ToUpper(key)); ok {
			values[key] = value
		}
	}

	addr, err := netip.ParseAddrPort(values["listen_addr"])
	if err != nil || addr.Port() == 0 || addr.Addr().Zone() != "" {
		return Config{}, errors.New("listen_addr requires an IP address and a port from 1 to 65535")
	}
	var level slog.Level
	switch strings.ToLower(values["log_level"]) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		return Config{}, errors.New("log_level must be debug, info, warn or error")
	}
	timeout, err := time.ParseDuration(values["shutdown_timeout"])
	if err != nil || timeout <= 0 || timeout > 30*time.Second {
		return Config{}, errors.New("shutdown_timeout must be positive and at most 30s")
	}
	return Config{ListenAddr: addr.String(), LogLevel: level, ShutdownTimeout: timeout}, nil
}

func readFile(path string, values map[string]string) error {
	file, err := os.Open(path)
	if err != nil {
		return errors.New("cannot open configuration file")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxFileSize+1))
	if err != nil || len(data) > maxFileSize {
		return errors.New("cannot read configuration file within 64 KiB limit")
	}
	var settings map[string]*string
	if err := json.Unmarshal(data, &settings); err != nil || settings == nil {
		return errors.New("configuration file must be a JSON object with string values")
	}
	for key, value := range settings {
		if _, known := values[key]; !known || value == nil {
			return errors.New("configuration file contains an unknown setting or null value")
		}
		values[key] = *value
	}
	return nil
}
