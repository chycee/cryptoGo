package domain

// Order represents a trading order.
// All monetary values are strictly int64.
type Order struct {
	ID           string
	Symbol       string
	Side         string // "BUY", "SELL"
	Type         string // "LIMIT", "MARKET"
	PriceMicros  int64  // Limit Price in Micros. 0 for Market Order.
	QtySats      int64  // Order Quantity in Satoshis.
	Status       string // "NEW", "PARTIALLY_FILLED", "FILLED", "CANCELED"
	CreatedUnixM int64  // Unix Microseconds
}

const (
	SideBuy  = "BUY"
	SideSell = "SELL"

	OrderTypeLimit  = "LIMIT"
	OrderTypeMarket = "MARKET"

	OrderStatusNew             = "NEW"
	OrderStatusPartiallyFilled = "PARTIALLY_FILLED"
	OrderStatusFilled          = "FILLED"
	OrderStatusCanceled        = "CANCELED"
)

// IsOpen checks if the order is still active.
func (o *Order) IsOpen() bool {
	return o.Status == "NEW" || o.Status == "PARTIALLY_FILLED"
}
