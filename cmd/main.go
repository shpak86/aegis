package main

import (
	"aegis/internal/captcha"
	"aegis/internal/config"
	"aegis/internal/fingerprint"
	"aegis/internal/limiter"
	"aegis/internal/middleware"
	"aegis/internal/server"
	"aegis/internal/sha_challenge"
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

func startServer(ctx context.Context, cancel context.CancelFunc, cfg *config.Config) *server.ApiServer {

	// Token manager
	var tokenManager usecase.TokenManager
	switch cfg.Verification.Type {
	case "js-challenge":
		m := sha_challenge.NewShaChallengeTokenManager(
			cfg.PermanentTokens,
			cfg.Verification.Complexity,
		)
		go m.Serve(ctx)
		tokenManager = m
	case "captcha":
		tokenManager = captcha.NewCaptchaTokenManager(
			ctx,
			cfg.PermanentTokens,
			cfg.Verification.Complexity,
		)
	default:
		slog.Error("Unknown verification type", "verification", cfg.Verification.Type)
		os.Exit(1)
	}

	// Rate limiter
	rateLimiter := limiter.NewRpsLimiter(ctx, tokenManager)
	var protections []usecase.Protection
	for i := range cfg.Protections {
		protection := usecase.Protection(cfg.Protections[i])
		protections = append(protections, protection)
		rateLimiter.AddLimit(protection)
	}
	go rateLimiter.Serve()

	// Fingerprint calculator
	fingerprintCalculator := fingerprint.NewRequestFingerprintCalculator()

	// Chain
	chain := middleware.NewChain(
		middleware.NewHttpFingerprintEnricher(fingerprintCalculator),
		middleware.NewPathProtector(fingerprintCalculator, rateLimiter, tokenManager, protections),
	)
	apiServer := server.NewApiServer(cfg.Address, chain, fingerprintCalculator, tokenManager)
	go func() {
		slog.Info("Serving API " + cfg.Address)
		err := apiServer.Serve()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server stopped abnormal")
		} else {
			slog.Info("Server stopped")
		}
		cancel()
	}()
	return apiServer
}

func main() {
	var err error

	versionProvider := version.NewVersionProvider("0.4.3")

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
	prepareLogger(cfg.Logger.Level)

	appCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	apiServer := startServer(appCtx, cancel, &cfg)
	<-appCtx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	apiServer.Shutdown(shutdownCtx)
}
