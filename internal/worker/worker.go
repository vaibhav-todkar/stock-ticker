// Package worker provides a goroutine-based polling worker for stock quotes.
package worker

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/vaibhav-todkar/stock-ticker/internal/model"
	"github.com/vaibhav-todkar/stock-ticker/internal/service"
)

// Worker polls the Finnhub API at a configured interval for multiple symbols.
type Worker struct {
	svc          service.StockService
	symbols      []string
	pollInterval time.Duration
	logger       *zap.Logger
	startTime    time.Time

	mu           sync.RWMutex
	latestQuotes map[string]*model.StockQuote
}

// New creates a new Worker.
func New(svc service.StockService, symbols []string, pollInterval time.Duration, logger *zap.Logger) *Worker {
	return &Worker{
		svc:          svc,
		symbols:      symbols,
		pollInterval: pollInterval,
		logger:       logger,
		startTime:    time.Now(),
		latestQuotes: make(map[string]*model.StockQuote),
	}
}

// Start begins the polling loop. It blocks until ctx is cancelled.
func (w *Worker) Start(ctx context.Context) {
	w.logger.Info("worker starting",
		zap.Strings("symbols", w.symbols),
		zap.Duration("poll_interval", w.pollInterval),
	)

	// Fetch immediately on start, then on every tick.
	w.fetchAll(ctx)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("worker stopping gracefully")
			return
		case <-ticker.C:
			w.fetchAll(ctx)
		}
	}
}

// fetchAll concurrently fetches all symbols, respecting a simple rate limiter.
func (w *Worker) fetchAll(ctx context.Context) {
	// Finnhub free tier: 60 req/min → minimum 1s between requests per goroutine.
	// Use a semaphore to cap concurrent requests.
	sem := make(chan struct{}, 10)

	var wg sync.WaitGroup
	for _, sym := range w.symbols {
		wg.Add(1)
		go func(symbol string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			q, err := w.svc.FetchQuote(ctx, symbol)
			if err != nil {
				w.logger.Error("failed to fetch quote",
					zap.String("symbol", symbol),
					zap.Error(err),
				)
				return
			}

			w.mu.Lock()
			w.latestQuotes[symbol] = q
			w.mu.Unlock()

			printQuote(q)
		}(sym)
	}

	wg.Wait()
}

// printQuote formats and prints a stock quote to stdout.
func printQuote(q *model.StockQuote) {
	sign := "+"
	if q.Change < 0 {
		sign = ""
	}
	fmt.Printf(
		"[%s] [INFO] Symbol: %-6s | Price: $%-10.2f | Change: %s%.2f (%s%.2f%%) | High: $%.2f | Low: $%.2f\n",
		q.FetchedAt.UTC().Format(time.RFC3339),
		q.Symbol,
		q.CurrentPrice,
		sign, q.Change,
		sign, q.ChangePercent,
		q.HighPrice,
		q.LowPrice,
	)
}

// LatestQuotes returns a snapshot of the most recently fetched quotes.
func (w *Worker) LatestQuotes() map[string]*model.StockQuote {
	w.mu.RLock()
	defer w.mu.RUnlock()

	snapshot := make(map[string]*model.StockQuote, len(w.latestQuotes))
	for k, v := range w.latestQuotes {
		snapshot[k] = v
	}
	return snapshot
}

// HealthInfo returns health check data for the /health endpoint.
func (w *Worker) HealthInfo() HealthResponse {
	uptime := time.Since(w.startTime).Round(time.Second).String()
	return HealthResponse{
		Status:         "ok",
		Uptime:         uptime,
		SymbolsTracked: len(w.symbols),
	}
}

// HealthResponse is the JSON body returned by the /health endpoint.
type HealthResponse struct {
	Status         string `json:"status"`
	Uptime         string `json:"uptime"`
	SymbolsTracked int    `json:"symbols_tracked"`
}

// ServeHealth starts an HTTP server on the given port serving GET /health.
// It blocks until ctx is cancelled.
func ServeHealth(ctx context.Context, port int, w *Worker, logger *zap.Logger) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		info := w.HealthInfo()
		rw.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(rw, `{"status":%q,"uptime":%q,"symbols_tracked":%d}`,
			info.Status, info.Uptime, info.SymbolsTracked)
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("health server shutdown error", zap.Error(err))
		}
	}()

	logger.Info("health check server starting", zap.Int("port", port))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("health server error", zap.Error(err))
	}
}
