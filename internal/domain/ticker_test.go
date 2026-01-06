package domain

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestMarketData_GapPct(t *testing.T) {
	t.Run("Normal Calculation", func(t *testing.T) {
		spot := Ticker{Price: decimal.NewFromInt(100)}
		future := Ticker{Price: decimal.NewFromInt(105)}

		data := MarketData{
			BitgetS: &spot,
			BitgetF: &future,
		}

		gap := data.GapPct()
		if gap == nil || !gap.Equal(decimal.NewFromInt(5)) {
			t.Errorf("Expected 5%%, got %v", gap)
		}
	})

	t.Run("Safety: Nil Pointers", func(t *testing.T) {
		data := MarketData{}
		if data.GapPct() != nil {
			t.Error("Should return nil when tickers are missing")
		}
	})

	t.Run("Safety: Zero Price", func(t *testing.T) {
		spot := Ticker{Price: decimal.Zero}
		future := Ticker{Price: decimal.NewFromInt(105)}
		data := MarketData{BitgetS: &spot, BitgetF: &future}
		if data.GapPct() != nil {
			t.Error("Should return nil when spot price is zero to avoid crash")
		}
	})
}

func TestMarketData_IsBreakoutHigh(t *testing.T) {
	t.Run("Breakout High Detect", func(t *testing.T) {
		high := decimal.NewFromInt(100)
		data := MarketData{
			Upbit: &Ticker{
				Price:          decimal.NewFromInt(101),
				HistoricalHigh: &high,
			},
		}

		if !data.IsBreakoutHigh() {
			t.Error("Should detect breakout high")
		}
	})
}
