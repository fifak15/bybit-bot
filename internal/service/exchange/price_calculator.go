package exchange

import (
	"bybit-bot/internal/model"
	"bybit-bot/internal/repository"
	"bybit-bot/internal/service/event"
	"log"
	"math"
)

type PriceCalculatorInterface interface {
	CalculateBuy(tradeLimit model.TradeLimits) model.BuyPrice
	CalculateSell(tradeLimit model.TradeLimits, order model.Order) (float64, error)
}

type PriceCalculator struct {
	OrderRepository  repository.OrderRepository
	WSListener       *event.WSListener
	WalletRepository *repository.WalletRepository
}

func (pc *PriceCalculator) CalculateQuantity(symbol string, entryPrice float64) float64 {
	balance, err := pc.WalletRepository.GetLatestWalletInfo("USDT")
	if err != nil {
		log.Printf("[Риск-менеджмент] Ошибка получения баланса: %v", err)
		return 0
	}

	maxPositionPercent := 0.10 // 10% от баланса
	positionSize := balance.WalletBalance * maxPositionPercent
	quantity := positionSize / entryPrice

	quantityRounded := math.Round(quantity*100) / 100

	log.Printf("[Риск-менеджмент] %s:\n"+
		"  Баланс: %.2f USDT\n"+
		"  Цена входа: %.2f\n"+
		"  Объем позиции: %.4f BTC\n"+
		"  Стоимость позиции: %.2f USDT",
		symbol,
		balance.WalletBalance,
		entryPrice,
		quantityRounded,
		quantityRounded*entryPrice)

	return quantityRounded
}
