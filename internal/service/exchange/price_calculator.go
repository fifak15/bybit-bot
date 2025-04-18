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

/*func (pc *PriceCalculator) CalculateBuy(tradeLimit model.TradeLimits) model.BuyPrice {
	lastKline, ok := pc.WSListener.GetKlineByTopic(tradeLimit.Symbol)

	if !ok || lastKline.Open == 0 || lastKline.Close == 0 {
		return model.BuyPrice{
			Price: 0.00,
			Error: errors.New(fmt.Sprintf("[%s] Current price is unknown, wait...", tradeLimit.Symbol)),
		}
	}

	return model.BuyPrice{
		Price: lastKline.Close,
		Error: nil,
	}
}

func (pc *PriceCalculator) CalculateSell(tradeLimit model.TradeLimits) (float64, error) {
	lastKline, ok := pc.WSListener.GetKlineByTopic(tradeLimit.Symbol)

	if !ok || lastKline.Open == 0 || lastKline.Close == 0 {
		return 0.00,
			errors.New(fmt.Sprintf("[%s] Current price is unknown, wait...", tradeLimit.Symbol))
	}
	currentPrice := lastKline.Close

	return currentPrice, nil
}*/

func (pc *PriceCalculator) CalculateQuantity(symbol string, entryPrice, stopLossPrice float64) float64 {
	existing, err := pc.WalletRepository.GetLatestWalletInfo("USDT")
	if err != nil {
		log.Printf("Ошибка получения данных баланса: %v", err)
		return 0.0
	}

	riskPerUnit := math.Abs(entryPrice - stopLossPrice)
	if riskPerUnit == 0 {
		log.Printf("Ошибка расчёта: разница между ценой входа и стоп‑лоссом равна нулю для %s", symbol)
		return 0.0
	}

	riskCapital := existing.WalletBalance * 0.01
	quantity := riskCapital / riskPerUnit

	quantityRounded := math.Round(quantity*100) / 100

	log.Printf("Для %s: availableFunds=%.2f, riskCapital=%.2f, riskPerUnit=%.2f, quantity=%.6f, roundedQuantity=%.2f",
		symbol, existing.WalletBalance, riskCapital, riskPerUnit, quantity, quantityRounded)
	return quantityRounded
}
