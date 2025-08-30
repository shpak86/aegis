package main

import (
	"aegis/internal/api"
	"aegis/internal/config"
	"aegis/internal/service"
	"aegis/internal/usecase"
	"aegis/internal/version"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func prepareConfiguration() (cfg config.Config, err error) {
	configPath := "/etc/aegis/config.json"
	content, err := os.ReadFile(configPath)
	if err != nil {
		return
	}
	err = json.Unmarshal(content, &cfg)
	return
}

func prepareLogger(level string) {
	opts := slog.HandlerOptions{}
	switch level {
	case "error":
		opts.Level = slog.LevelError
	case "warn":
		opts.Level = slog.LevelWarn
	case "debug":
		opts.Level = slog.LevelDebug
	default:
		opts.Level = slog.LevelInfo
	}
	appLogger := slog.NewJSONHandler(os.Stderr, &opts)
	slog.SetDefault(slog.New(appLogger))
}

func prepareChainer(ctx context.Context, cfg config.Config) (chainer *service.Chainer) {
	protections := []usecase.Protection{}
	for _, configProtection := range cfg.Protections {
		protections = append(protections, usecase.Protection{Path: configProtection.Path, Method: configProtection.Method, Limit: configProtection.Limit})
	}

	chainer = service.BasicAntibotChainer(ctx, cfg.Token.Complexity, protections)
	return
}

func main() {
	var err error

	versionProvider := version.NewVersionProvider("0.1.0")

	versionFlag := flag.Bool("version", false, "Print Aegis version")
	flag.Parse()
	if *versionFlag {
		fmt.Printf("Aegis %s", versionProvider.String())
		return
	}

	var cfg config.Config
	if cfg, err = prepareConfiguration(); err != nil {
		slog.Error("Failed to prepare app config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	prepareLogger(cfg.Logger.Level)
	chainer := prepareChainer(appCtx, cfg)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	serverError := make(chan error, 1)

	server := api.NewApiServer(cfg.Address, chainer)
	go func() {
		slog.Debug("API address", slog.String("address", cfg.Address))
		err = server.Serve()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverError <- err
		} else {
			slog.Info("Server stopped")
		}
	}()
	select {
	case err := <-serverError:
		slog.Error("API server", slog.String("error", err.Error()))
	case <-sigChan:
		slog.Info("Shutting down server")
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Shutdown server", slog.String("error", err.Error()))
		return
	}
}
