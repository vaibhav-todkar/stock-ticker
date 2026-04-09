// Package client provides a Finnhub HTTP client with retry and backoff support.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/vaibhav-todkar/stock-ticker/internal/model"
)

// StockClient defines the interface for fetching stock quotes.
type StockClient interface {
	GetQuote(ctx context.Context, symbol string) (*model.StockQuote, error)
}

// finnhubClient implements StockClient using the Finnhub REST API.
type finnhubClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	maxRetries int
	logger     *zap.Logger
}

// ClientOption is a functional option for configuring finnhubClient.
type ClientOption func(*finnhubClient)

// WithLogger sets the logger.
func WithLogger(l *zap.Logger) ClientOption {
	return func(c *finnhubClient) { c.logger = l }
}

// New creates a new Finnhub StockClient.
func New(baseURL, apiKey string, timeout time.Duration, maxRetries int, opts ...ClientOption) StockClient {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	c := &finnhubClient{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
		baseURL:    baseURL,
		apiKey:     apiKey,
		maxRetries: maxRetries,
		logger:     zap.NewNop(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// GetQuote fetches a stock quote for the given symbol from Finnhub.
// It retries up to maxRetries times with exponential backoff on transient errors.
func (c *finnhubClient) GetQuote(ctx context.Context, symbol string) (*model.StockQuote, error) {
	url := fmt.Sprintf("%s/quote?symbol=%s&token=%s", c.baseURL, symbol, c.apiKey)

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second // 1s, 2s, 4s …
			c.logger.Debug("retrying request",
				zap.String("symbol", symbol),
				zap.Int("attempt", attempt),
				zap.Duration("backoff", backoff),
			)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		start := time.Now()
		quote, err := c.doRequest(ctx, url, symbol)
		latency := time.Since(start)

		if err == nil {
			c.logger.Debug("fetched quote",
				zap.String("symbol", symbol),
				zap.Float64("price", quote.CurrentPrice),
				zap.Duration("latency", latency),
			)
			return quote, nil
		}

		if isPermanentError(err) {
			return nil, err
		}

		lastErr = err
		c.logger.Warn("transient error fetching quote",
			zap.String("symbol", symbol),
			zap.Int("attempt", attempt),
			zap.Error(err),
		)
	}

	return nil, fmt.Errorf("all %d retries exhausted for symbol %s: %w", c.maxRetries, symbol, lastErr)
}

func (c *finnhubClient) doRequest(ctx context.Context, url, symbol string) (*model.StockQuote, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, permanentError{fmt.Errorf("building request: %w", err)}
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate limited (429)")
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, permanentError{fmt.Errorf("authentication error (%d)", resp.StatusCode)}
	case resp.StatusCode >= 500:
		return nil, fmt.Errorf("server error (%d)", resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return nil, permanentError{fmt.Errorf("unexpected status code %d", resp.StatusCode)}
	}

	var raw model.FinnhubQuoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, permanentError{fmt.Errorf("decoding response: %w", err)}
	}

	now := time.Now()
	change := raw.C - raw.PC
	var changePct float64
	if raw.PC != 0 {
		changePct = (change / raw.PC) * 100
	}

	return &model.StockQuote{
		Symbol:        symbol,
		CurrentPrice:  raw.C,
		HighPrice:     raw.H,
		LowPrice:      raw.L,
		OpenPrice:     raw.O,
		PreviousClose: raw.PC,
		Change:        change,
		ChangePercent: changePct,
		Timestamp:     time.Unix(raw.T, 0),
		FetchedAt:     now,
	}, nil
}

// permanentError wraps an error that should not be retried.
type permanentError struct{ cause error }

func (e permanentError) Error() string { return e.cause.Error() }
func (e permanentError) Unwrap() error { return e.cause }

// isPermanentError returns true if the error is permanent and should not be retried.
func isPermanentError(err error) bool {
	var pe permanentError
	return isErrorType(err, &pe)
}

func isErrorType(err error, target interface{}) bool {
	switch target.(type) {
	case *permanentError:
		_, ok := err.(permanentError)
		return ok
	}
	return false
}
