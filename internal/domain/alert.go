package domain

import "github.com/shopspring/decimal"

// AlertConfig represents a price alert configuration
type AlertConfig struct {
	Symbol       string          `json:"symbol"`
	TargetPrice  decimal.Decimal `json:"target"`
	Direction    string          `json:"direction"` // "UP" or "DOWN"
	Exchange     string          `json:"exchange"`  // "UPBIT", "BITGET_F"
	IsPersistent bool            `json:"is_persistent"`
	active       bool
}

// NewAlertConfig creates a new alert configuration.
// Direction is automatically determined based on currentPrice:
// - UP: targetPrice > currentPrice (waiting for price to rise)
// - DOWN: targetPrice < currentPrice (waiting for price to fall)
func NewAlertConfig(symbol string, targetPrice, currentPrice decimal.Decimal, exchange string, isPersistent bool) *AlertConfig {
	direction := "UP"
	if targetPrice.LessThan(currentPrice) {
		direction = "DOWN"
	}
	return &AlertConfig{
		Symbol:       symbol,
		TargetPrice:  targetPrice,
		Direction:    direction,
		Exchange:     exchange,
		IsPersistent: isPersistent,
		active:       true,
	}
}

// IsActive returns whether the alert is active
func (a *AlertConfig) IsActive() bool {
	return a.active
}

// SetActive sets the alert's active state
func (a *AlertConfig) SetActive(active bool) {
	a.active = active
}

// CheckCondition checks if alert condition is met.
// Returns true when:
// - Direction is UP and currentPrice >= targetPrice
// - Direction is DOWN and currentPrice <= targetPrice
func (a *AlertConfig) CheckCondition(currentPrice decimal.Decimal) bool {
	if !a.active {
		return false
	}
	switch a.Direction {
	case "UP":
		return currentPrice.GreaterThanOrEqual(a.TargetPrice)
	case "DOWN":
		return currentPrice.LessThanOrEqual(a.TargetPrice)
	default:
		return false
	}
}

