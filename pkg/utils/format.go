// Package utils provides shared utility functions for the stock ticker service.
package utils

import "fmt"

// FormatPrice formats a float64 as a USD price string.
func FormatPrice(price float64) string {
	return fmt.Sprintf("$%.2f", price)
}

// FormatChange formats a price change value with a +/- sign.
func FormatChange(change float64) string {
	if change >= 0 {
		return fmt.Sprintf("+%.2f", change)
	}
	return fmt.Sprintf("%.2f", change)
}

// FormatChangePercent formats a percentage change with a +/- sign.
func FormatChangePercent(pct float64) string {
	if pct >= 0 {
		return fmt.Sprintf("+%.2f%%", pct)
	}
	return fmt.Sprintf("%.2f%%", pct)
}
