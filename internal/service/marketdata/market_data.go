package marketdata

import "bybit-bot/internal/model"

type Service interface {
	GetRecentKlines(symbol, interval string, n int) ([]model.KlineData, bool)
}
