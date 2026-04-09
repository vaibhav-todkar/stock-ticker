package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/vaibhav-todkar/stock-ticker/internal/model"
	"github.com/vaibhav-todkar/stock-ticker/internal/service"
)

// mockClient is a test double for client.StockClient.
type mockClient struct {
	quotes map[string]*model.StockQuote
	err    error
}

func (m *mockClient) GetQuote(_ context.Context, symbol string) (*model.StockQuote, error) {
	if m.err != nil {
		return nil, m.err
	}
	if q, ok := m.quotes[symbol]; ok {
		return q, nil
	}
	return nil, errors.New("symbol not found: " + symbol)
}

func newTestQuote(symbol string, price, prev float64) *model.StockQuote {
	change := price - prev
	var pct float64
	if prev != 0 {
		pct = (change / prev) * 100
	}
	return &model.StockQuote{
		Symbol:        symbol,
		CurrentPrice:  price,
		HighPrice:     price + 1,
		LowPrice:      price - 1,
		OpenPrice:     prev,
		PreviousClose: prev,
		Change:        change,
		ChangePercent: pct,
		Timestamp:     time.Now(),
		FetchedAt:     time.Now(),
	}
}

func TestFetchQuote_Success(t *testing.T) {
	mc := &mockClient{
		quotes: map[string]*model.StockQuote{
			"AAPL": newTestQuote("AAPL", 182.50, 180.15),
		},
	}
	svc := service.New(mc, zap.NewNop())

	q, err := svc.FetchQuote(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if q.Symbol != "AAPL" {
		t.Errorf("expected symbol AAPL, got %s", q.Symbol)
	}
	if q.CurrentPrice != 182.50 {
		t.Errorf("expected price 182.50, got %f", q.CurrentPrice)
	}
	expectedChange := 182.50 - 180.15
	if abs(q.Change-expectedChange) > 1e-6 {
		t.Errorf("expected change %f, got %f", expectedChange, q.Change)
	}
}

func TestFetchQuote_EmptySymbol(t *testing.T) {
	svc := service.New(&mockClient{}, zap.NewNop())

	_, err := svc.FetchQuote(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty symbol")
	}
}

func TestFetchQuote_ClientError(t *testing.T) {
	mc := &mockClient{err: errors.New("network error")}
	svc := service.New(mc, zap.NewNop())

	_, err := svc.FetchQuote(context.Background(), "AAPL")
	if err == nil {
		t.Fatal("expected error from client")
	}
}

func TestFetchMultipleQuotes_Success(t *testing.T) {
	mc := &mockClient{
		quotes: map[string]*model.StockQuote{
			"AAPL":  newTestQuote("AAPL", 182.50, 180.15),
			"GOOGL": newTestQuote("GOOGL", 141.80, 142.25),
			"MSFT":  newTestQuote("MSFT", 310.00, 308.50),
		},
	}
	svc := service.New(mc, zap.NewNop())

	quotes, err := svc.FetchMultipleQuotes(context.Background(), []string{"AAPL", "GOOGL", "MSFT"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(quotes) != 3 {
		t.Errorf("expected 3 quotes, got %d", len(quotes))
	}
}

func TestFetchMultipleQuotes_PartialFailure(t *testing.T) {
	mc := &mockClient{
		quotes: map[string]*model.StockQuote{
			"AAPL": newTestQuote("AAPL", 182.50, 180.15),
			// BADTICKER not present → will return error
		},
	}
	svc := service.New(mc, zap.NewNop())

	quotes, err := svc.FetchMultipleQuotes(context.Background(), []string{"AAPL", "BADTICKER"})
	// Should still return the successful quote, not error out entirely.
	if err != nil {
		t.Fatalf("expected no error for partial failure, got: %v", err)
	}
	if len(quotes) != 1 {
		t.Errorf("expected 1 successful quote, got %d", len(quotes))
	}
}

func TestFetchMultipleQuotes_AllFail(t *testing.T) {
	mc := &mockClient{err: errors.New("all fail")}
	svc := service.New(mc, zap.NewNop())

	_, err := svc.FetchMultipleQuotes(context.Background(), []string{"AAPL", "GOOGL"})
	if err == nil {
		t.Fatal("expected error when all fetches fail")
	}
}

func TestFetchMultipleQuotes_EmptySymbols(t *testing.T) {
	svc := service.New(&mockClient{}, zap.NewNop())

	_, err := svc.FetchMultipleQuotes(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error for empty symbols list")
	}
}

func TestChangeCalculation(t *testing.T) {
	q := newTestQuote("TEST", 110.0, 100.0)
	if abs(q.Change-10.0) > 1e-6 {
		t.Errorf("expected change 10.0, got %f", q.Change)
	}
	if abs(q.ChangePercent-10.0) > 1e-6 {
		t.Errorf("expected change percent 10.0, got %f", q.ChangePercent)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
