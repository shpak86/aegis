package main

import (
	"aegis/internal/config"
	"aegis/internal/server"
	"aegis/internal/usecase"
	"aegis/internal/version"
	"context"
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

func prepareLogger(level string) {
	opts := slog.HandlerOptions{}
	switch level {
	case "ERROR":
		opts.Level = slog.LevelError
	case "WARNING":
		opts.Level = slog.LevelWarn
	case "INFO":
		opts.Level = slog.LevelInfo
	case "DEBUG":
		opts.Level = slog.LevelDebug
	default:
		slog.Error("Failed to prepare logger", slog.String("error", fmt.Sprintf("unknown level %s", level)))
		os.Exit(1)
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &opts)))
}

func main() {
	var err error

	versionProvider := version.NewVersionProvider("0.2.1")

	versionFlag := flag.Bool("version", false, "Print Aegis version")
	configPath := flag.String("config", "/etc/aegis/config.json", "Configuration path")
	flag.Parse()
	if *versionFlag {
		fmt.Printf("Aegis %s", versionProvider.String())
		return
	}

	var cfg config.Config
	if err = cfg.Load(*configPath); err != nil {
		slog.Error("Failed to prepare app config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	prepareLogger(cfg.Logger.Level)
	protections := make([]usecase.Protection, len(cfg.Protections))
	for i := range cfg.Protections {
		protections[i] = usecase.Protection(cfg.Protections[i])
	}

	slog.Info("Starting Aegis server")

	server := server.NewServer(
		appCtx,
		cfg.Address,
		protections,
		cfg.Verification.Type,
		cfg.Verification.Complexity,
	)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	serverError := make(chan error, 1)

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
