package domain

import "crypto_go/pkg/quant"

// MarketState holds the current state of a single market.
// Fields are ordered for cache-line efficiency: hot fields (price/qty) first.
// Moved from engine/sequencer.go to avoid circular dependency.
type MarketState struct {
	// Hot fields (frequently accessed together in the hotpath)
	PriceMicros     quant.PriceMicros `json:"price"`
	TotalQtySats    quant.QtySats     `json:"qty"`
	LastUpdateUnixM quant.TimeStamp   `json:"last_update"`
	// Cold fields (less frequent access)
	Symbol string `json:"symbol"`
}
