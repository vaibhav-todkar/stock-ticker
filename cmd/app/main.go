// Command app is the entry point for the stock ticker microservice.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/vaibhav-todkar/stock-ticker/internal/client"
	"github.com/vaibhav-todkar/stock-ticker/internal/config"
	"github.com/vaibhav-todkar/stock-ticker/internal/logger"
	"github.com/vaibhav-todkar/stock-ticker/internal/service"
	"github.com/vaibhav-todkar/stock-ticker/internal/worker"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration.
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Initialise logger.
	log, err := logger.New(cfg.Logging.Level, cfg.Logging.Format)
	if err != nil {
		// Fall back to a no-op logger; log level errors are non-fatal.
		log, _ = logger.New("info", "json")
	}
	defer log.Sync()

	if cfg.API.APIKey == "" {
		return fmt.Errorf("FINNHUB_API_KEY environment variable is not set")
	}

	log.Info("starting stock ticker service",
		zap.Strings("symbols", cfg.API.Symbols),
		zap.Duration("poll_interval", cfg.API.PollInterval),
		zap.Int("server_port", cfg.Server.Port),
	)

	// Build client and service.
	stockClient := client.New(
		cfg.API.BaseURL,
		cfg.API.APIKey,
		cfg.API.Timeout,
		cfg.API.MaxRetries,
		client.WithLogger(log.Logger),
	)
	svc := service.New(stockClient, log.Logger)

	// Build worker.
	w := worker.New(svc, cfg.API.Symbols, cfg.API.PollInterval, log.Logger)

	// Root context with graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start health check server in a separate goroutine.
	go worker.ServeHealth(ctx, cfg.Server.Port, w, log.Logger)

	// Start polling worker — blocks until ctx is cancelled.
	w.Start(ctx)

	log.Info("stock ticker service stopped")
	return nil
}
