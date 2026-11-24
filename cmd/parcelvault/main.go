package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lfreixial/ParcelVault/internal/api"
	"github.com/lfreixial/ParcelVault/internal/config"
	"github.com/lfreixial/ParcelVault/internal/database"
	"github.com/lfreixial/ParcelVault/internal/server"
)

func main() {
	var level slog.LevelVar
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: &level}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, logger, &level); err != nil {
		logger.Error("application stopped", "reason", err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger, level *slog.LevelVar) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	level.Set(cfg.LogLevel)
	startup, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	db, err := database.Open(startup, cfg.DatabaseDriver, cfg.DatabaseSource)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := database.Migrate(startup, db, cfg.DatabaseDriver); err != nil {
		return err
	}
	cancel()
	ln, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		return errors.New("cannot bind HTTP listener")
	}
	handler := api.New(func(request context.Context) bool {
		probe, cancel := context.WithTimeout(request, time.Second)
		defer cancel()
		return ctx.Err() == nil && db.PingContext(probe) == nil
	})
	return server.Run(ctx, ln, handler, cfg.ShutdownTimeout, logger)
}
