// Package service provides the business logic for fetching and normalizing stock quotes.
package service

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/vaibhav-todkar/stock-ticker/internal/client"
	"github.com/vaibhav-todkar/stock-ticker/internal/model"
)

// StockService defines the interface for the stock data service.
type StockService interface {
	FetchQuote(ctx context.Context, symbol string) (*model.StockQuote, error)
	FetchMultipleQuotes(ctx context.Context, symbols []string) ([]*model.StockQuote, error)
}

// stockService implements StockService.
type stockService struct {
	client client.StockClient
	logger *zap.Logger
}

// New creates a new StockService.
func New(c client.StockClient, logger *zap.Logger) StockService {
	return &stockService{
		client: c,
		logger: logger,
	}
}

// FetchQuote fetches and returns a normalized stock quote for the given symbol.
func (s *stockService) FetchQuote(ctx context.Context, symbol string) (*model.StockQuote, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol must not be empty")
	}

	quote, err := s.client.GetQuote(ctx, symbol)
	if err != nil {
		s.logger.Error("failed to fetch quote",
			zap.String("symbol", symbol),
			zap.Error(err),
		)
		return nil, fmt.Errorf("fetching quote for %s: %w", symbol, err)
	}

	s.logger.Info("fetched quote",
		zap.String("symbol", quote.Symbol),
		zap.Float64("price", quote.CurrentPrice),
		zap.Float64("change", quote.Change),
		zap.Float64("change_percent", quote.ChangePercent),
	)

	return quote, nil
}

// FetchMultipleQuotes concurrently fetches quotes for all provided symbols.
// Errors for individual symbols are logged but do not stop other fetches.
func (s *stockService) FetchMultipleQuotes(ctx context.Context, symbols []string) ([]*model.StockQuote, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("symbols list must not be empty")
	}

	type result struct {
		quote *model.StockQuote
		err   error
	}

	results := make([]result, len(symbols))
	var wg sync.WaitGroup

	for i, sym := range symbols {
		wg.Add(1)
		go func(idx int, symbol string) {
			defer wg.Done()
			q, err := s.FetchQuote(ctx, symbol)
			results[idx] = result{quote: q, err: err}
		}(i, sym)
	}

	wg.Wait()

	quotes := make([]*model.StockQuote, 0, len(symbols))
	var errs []error
	for _, r := range results {
		if r.err != nil {
			errs = append(errs, r.err)
			continue
		}
		quotes = append(quotes, r.quote)
	}

	if len(errs) > 0 && len(quotes) == 0 {
		return nil, fmt.Errorf("all quote fetches failed: %v", errs)
	}

	return quotes, nil
}
