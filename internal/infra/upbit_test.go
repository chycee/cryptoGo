package infra

import (
	"testing"
)

// =====================================================
// Upbit Precision Tests
// =====================================================

func TestDeterminePrecision(t *testing.T) {
	tests := []struct {
		name  string
		price float64
		want  int
	}{
		{"high price >= 1000", 50000.0, 0},
		{"price >= 100", 500.0, 1},
		{"price >= 10", 50.0, 2},
		{"price >= 1", 5.0, 3},
		{"price >= 0.1", 0.5, 4},
		{"very small price", 0.05, 8},
		{"exact 1000", 1000.0, 0},
		{"exact 100", 100.0, 1},
		{"exact 10", 10.0, 2},
		{"exact 1", 1.0, 3},
		{"exact 0.1", 0.1, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determinePrecision(tt.price)
			if got != tt.want {
				t.Errorf("determinePrecision(%f) = %d, want %d", tt.price, got, tt.want)
			}
		})
	}
}
