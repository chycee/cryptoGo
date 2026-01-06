package domain

import "github.com/shopspring/decimal"

// Ticker represents price data from a single exchange
type Ticker struct {
	Symbol     string          `json:"symbol"`      // Unified symbol (e.g., "BTC")
	Price      decimal.Decimal `json:"price"`       // Current price
	Volume     decimal.Decimal `json:"volume"`      // 24h volume
	ChangeRate decimal.Decimal `json:"change_rate"` // 24h change (%)
	Exchange   string          `json:"exchange"`    // "UPBIT", "BITGET_S", "BITGET_F"
	Precision  int             `json:"precision"`   // Decimal places from exchange

	// Bitget Futures specific
	FundingRate     *decimal.Decimal `json:"funding_rate,omitempty"`
	NextFundingTime *int64           `json:"next_funding_time,omitempty"`
	HistoricalHigh  *decimal.Decimal `json:"historical_high,omitempty"`
	HistoricalLow   *decimal.Decimal `json:"historical_low,omitempty"`
}

// MarketData aggregates data for a single symbol from all exchanges
type MarketData struct {
	Symbol     string           `json:"symbol"`
	Upbit      *Ticker          `json:"upbit,omitempty"`
	BitgetS    *Ticker          `json:"bitget_s,omitempty"`
	BitgetF    *Ticker          `json:"bitget_f,omitempty"`
	Premium    *decimal.Decimal `json:"premium,omitempty"`
	StatusMsg  string           `json:"status_msg"`
	IsFavorite bool             `json:"is_favorite"`
}

// GapPct calculates Futures vs Spot gap percentage: 100 * (Future - Spot) / Spot
func (m *MarketData) GapPct() *decimal.Decimal {
	if m.BitgetS == nil || m.BitgetF == nil {
		return nil
	}
	if m.BitgetS.Price.IsZero() {
		return nil
	}

	gap := m.BitgetF.Price.Sub(m.BitgetS.Price).Div(m.BitgetS.Price).Mul(decimal.NewFromInt(100))
	return &gap
}

// IsBreakoutHigh returns true if price >= historical high
func (m *MarketData) IsBreakoutHigh() bool {
	if m.Upbit == nil || m.Upbit.HistoricalHigh == nil {
		return false
	}
	return m.Upbit.Price.GreaterThanOrEqual(*m.Upbit.HistoricalHigh)
}

// IsBreakoutLow returns true if price <= historical low
func (m *MarketData) IsBreakoutLow() bool {
	if m.Upbit == nil || m.Upbit.HistoricalLow == nil {
		return false
	}
	return m.Upbit.Price.LessThanOrEqual(*m.Upbit.HistoricalLow)
}

// BreakoutState returns "high", "low", or "normal"
func (m *MarketData) BreakoutState() string {
	if m.IsBreakoutHigh() {
		return "high"
	}
	if m.IsBreakoutLow() {
		return "low"
	}
	return "normal"
}

// ChangeDirection returns "positive", "negative", or "neutral"
func (m *MarketData) ChangeDirection() string {
	if m.Upbit == nil {
		return "neutral"
	}
	if m.Upbit.ChangeRate.IsPositive() {
		return "positive"
	}
	if m.Upbit.ChangeRate.IsNegative() {
		return "negative"
	}
	return "neutral"
}

