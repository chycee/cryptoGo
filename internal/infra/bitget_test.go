package infra

import (
	"testing"
)

// =====================================================
// Bitget Precision Tests
// =====================================================

func TestDetermineBitgetPrecision(t *testing.T) {
	tests := []struct {
		name     string
		priceStr string
		want     int
	}{
		{"integer", "12345", 0},
		{"one decimal", "123.4", 1},
		{"two decimals", "12.34", 2},
		{"four decimals", "0.1234", 4},
		{"eight decimals", "0.12345678", 8},
		{"zero", "0", 0},
		{"zero with decimal", "0.00", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineBitgetPrecision(tt.priceStr)
			if got != tt.want {
				t.Errorf("determineBitgetPrecision(%q) = %d, want %d", tt.priceStr, got, tt.want)
			}
		})
	}
}

func TestCalculateBitgetBackoff(t *testing.T) {
	tests := []struct {
		retryCount int
		minDelay   int64 // milliseconds
		maxDelay   int64 // milliseconds
	}{
		{0, 1000, 1000},     // 1s
		{1, 2000, 2000},     // 2s
		{2, 4000, 4000},     // 4s
		{3, 8000, 8000},     // 8s
		{10, 60000, 60000},  // max 60s
		{100, 60000, 60000}, // still max 60s
	}

	for _, tt := range tests {
		delay := calculateBitgetBackoff(tt.retryCount)
		delayMs := delay.Milliseconds()
		if delayMs < tt.minDelay || delayMs > tt.maxDelay {
			t.Errorf("calculateBitgetBackoff(%d) = %dms, want between %d and %d",
				tt.retryCount, delayMs, tt.minDelay, tt.maxDelay)
		}
	}
}

