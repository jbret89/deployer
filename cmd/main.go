package main

import (
	"errors"
	"log/slog"
	"os"

	"deployer/internal/config"
	httpapi "deployer/internal/http"
	"deployer/internal/service"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Error("failed to load .env file", "error", err)
		os.Exit(1)
	}

	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	deployerService := service.NewDeployer(cfg)
	server := httpapi.NewServer(cfg, logger, deployerService)

	logger.Info("starting deployer", "address", cfg.Address())

	if err := server.ListenAndServe(); err != nil {
		logger.Error("server exited", "error", err)
		os.Exit(1)
	}
}
