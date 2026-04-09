// Package model defines the data structures used throughout the stock ticker service.
package model

import "time"

// FinnhubQuoteResponse represents the raw JSON response from the Finnhub quote API.
type FinnhubQuoteResponse struct {
	C  float64 `json:"c"`  // Current price
	H  float64 `json:"h"`  // High price of the day
	L  float64 `json:"l"`  // Low price of the day
	O  float64 `json:"o"`  // Open price of the day
	PC float64 `json:"pc"` // Previous close price
	T  int64   `json:"t"`  // Timestamp (Unix)
}

// StockQuote represents a normalized stock quote with calculated fields.
type StockQuote struct {
	Symbol        string
	CurrentPrice  float64
	HighPrice     float64
	LowPrice      float64
	OpenPrice     float64
	PreviousClose float64
	Change        float64   // CurrentPrice - PreviousClose
	ChangePercent float64   // (Change / PreviousClose) * 100
	Timestamp     time.Time // timestamp from the API response
	FetchedAt     time.Time // when we fetched this data
}
