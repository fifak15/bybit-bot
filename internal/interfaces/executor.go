package interfaces

type Executor interface {
	PlaceLimitOrder(symbol, side string, price, qty, sl, tp float64) error
}
