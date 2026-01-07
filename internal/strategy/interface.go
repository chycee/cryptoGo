package strategy

import (
	"crypto_go/internal/domain"
	"crypto_go/pkg/quant"
)

// ActionType defines the type of trading action
type ActionType int

const (
	ActionBuy  ActionType = iota + 1
	ActionSell // Sell
)

// String returns the string representation of ActionType
func (a ActionType) String() string {
	switch a {
	case ActionBuy:
		return "BUY"
	case ActionSell:
		return "SELL"
	default:
		return "UNKNOWN"
	}
}

// Action represents a decision made by the strategy
type Action struct {
	Type   ActionType
	Symbol string
	Price  quant.PriceMicros
	Qty    quant.QtySats
}

// Strategy is the interface that all trading strategies must implement.
// It is called synchronously by the Sequencer.
type Strategy interface {
	// OnMarketUpdate is called when a market data update is received.
	// It returns a list of Actions to be executed.
	OnMarketUpdate(state domain.MarketState) []Action
}
