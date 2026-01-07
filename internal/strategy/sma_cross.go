package strategy

import (
	"crypto_go/internal/domain"
	"crypto_go/pkg/quant"
	"crypto_go/pkg/safe"
)

// SMACrossStrategy implements a simple SMA Crossover strategy.
// It is stateful and deterministic.
// OPTIMIZED: Uses a Ring Buffer to ensure Zero-Alloc in the hotpath.
type SMACrossStrategy struct {
	symbol      string
	shortPeriod int
	longPeriod  int

	// State (Ring Buffer)
	prices []int64
	head   int   // Current write position
	count  int   // Number of elements filled
	sum    int64 // Running sum for the longest period (optimization)

	prevShortSMA int64
	prevLongSMA  int64
}

// NewSMACrossStrategy creates a new instance.
func NewSMACrossStrategy(symbol string, shortPeriod, longPeriod int) *SMACrossStrategy {
	if shortPeriod >= longPeriod {
		panic("SMACrossStrategy: shortPeriod must be less than longPeriod")
	}
	return &SMACrossStrategy{
		symbol:      symbol,
		shortPeriod: shortPeriod,
		longPeriod:  longPeriod,
		prices:      make([]int64, longPeriod), // Fixed size allocation
		head:        0,
		count:       0,
		sum:         0,
	}
}

// OnMarketUpdate processes market updates and generates signals.
func (s *SMACrossStrategy) OnMarketUpdate(state domain.MarketState) []Action {
	// 1. Filter by symbol
	if state.Symbol != s.symbol {
		return nil
	}

	currentPrice := int64(state.PriceMicros)

	// 2. Update Price History (Ring Buffer)
	// If full, subtract the oldest value from sum before overwriting
	if s.count == s.longPeriod {
		oldestPrice := s.prices[s.head] // s.head points to the oldest value when full
		s.sum = safe.SafeSub(s.sum, oldestPrice)
	}

	// Add new price
	s.prices[s.head] = currentPrice
	s.sum = safe.SafeAdd(s.sum, currentPrice)

	// Move head
	s.head = (s.head + 1) % s.longPeriod

	// Increment count if not yet full
	if s.count < s.longPeriod {
		s.count++
	}

	// 3. Check if we have enough data
	if s.count < s.longPeriod {
		return nil
	}

	// 4. Calculate SMAs
	// Long SMA is easy: s.sum / s.longPeriod
	currLongSMA := safe.SafeDiv(s.sum, int64(s.longPeriod))

	// Short SMA requires manual calculation over the ring buffer
	currShortSMA := s.calculateShortSMA()

	var actions []Action

	// 5. Check for Cross
	if s.prevShortSMA != 0 && s.prevLongSMA != 0 {
		// Golden Cross: Short goes above Long
		if s.prevShortSMA <= s.prevLongSMA && currShortSMA > currLongSMA {
			actions = append(actions, Action{
				Type:   ActionBuy,
				Symbol: s.symbol,
				Price:  state.PriceMicros,
				Qty:    quant.QtySats(10000),
			})
		}

		// Dead Cross: Short goes below Long
		if s.prevShortSMA >= s.prevLongSMA && currShortSMA < currLongSMA {
			actions = append(actions, Action{
				Type:   ActionSell,
				Symbol: s.symbol,
				Price:  state.PriceMicros,
				Qty:    quant.QtySats(10000),
			})
		}
	}

	// 6. Update State
	s.prevShortSMA = currShortSMA
	s.prevLongSMA = currLongSMA

	return actions
}

// calculateShortSMA calculates the SMA for the short period using the ring buffer.
func (s *SMACrossStrategy) calculateShortSMA() int64 {
	var sum int64 = 0
	// Walk backwards from current head (which points to next write slot, so head-1 is latest)
	idx := s.head
	for i := 0; i < s.shortPeriod; i++ {
		idx--
		if idx < 0 {
			idx = s.longPeriod - 1
		}
		sum = safe.SafeAdd(sum, s.prices[idx])
	}
	return safe.SafeDiv(sum, int64(s.shortPeriod))
}
