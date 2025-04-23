package strategy

import (
	"bybit-bot/internal/client"
	"bybit-bot/internal/interfaces"
	"bybit-bot/internal/model"
	"bybit-bot/internal/repository"
	"bybit-bot/internal/service/account"
	"bybit-bot/internal/service/event"
	"bybit-bot/internal/service/exchange"
	"bybit-bot/internal/utils"
	"log"
	"strings"
	"time"
)

type VPAScalping struct {
	OrderRepository  repository.OrderRepository
	WalletRepository *repository.WalletRepository
	Formatter        *utils.Formatter
	BalanceService   *account.BalanceService
	PriceCalculator  *exchange.PriceCalculator
	MarketData       interfaces.Service
	Bybit            *client.ByBit
	WSListener       *event.WSListener
	StopLossPercent  float64
	RiskMang         *exchange.R
	SignalDetector   *SignalDetector
	Trading          interfaces.Executor
}

// Параметры стратегии
const (
	VolumeWindow   = 18 // число свечей для расчёта среднего объёма
	LookbackPeriod = 10 // число предыдущих свечей для оценки локального минимума/максимума
)
const (
	DefaultStopLossPct   = 0.00050 // 0.05%
	DefaultTakeProfitPct = 0.0015  // 0.15%
)

func (s *VPAScalping) Make(symbol, category string) {
	/*if !s.IsTradingTime() {
		log.Printf("Вне торгового времени: пропускаем %s", symbol)
		return
	}*/
	openOrders, err := s.Bybit.GetOpenOrders(category, symbol)
	if err != nil {
		log.Printf("Ошибка получения открытых ордеров для %s: %v", symbol, err)
		return
	}
	if openOrders != nil && len(openOrders.Orders) > 0 {
		log.Printf("Пропускаем %s: ошибка=%v, открытых ордеров=%d", symbol, err, len(openOrders.Orders))
		return
	}

	klines, ok := s.MarketData.GetRecentKlines(symbol, "1", VolumeWindow+LookbackPeriod)
	log.Printf("asdasdadadadad %s", klines)
	if !ok {
		log.Printf("Недостаточно данных свечей для %s", symbol)
		return
	}

	topicOrderbook := "orderbook.50." + strings.ToUpper(symbol)
	orderBook, ok := s.WSListener.GetOrderbookByTopic(topicOrderbook)
	if !ok || len(orderBook.Bids) == 0 || len(orderBook.Asks) == 0 {
		log.Printf("Недостаточно данных ордербука для %s", symbol)
		return
	}

	isLong := s.SignalDetector.CheckLongSignal(klines)
	log.Printf("isLong %s", isLong)
	isShort := s.SignalDetector.CheckShortSignal(klines)
	log.Printf("isShort %s", isShort)
	if !isLong && !isShort {
		log.Printf("Нет сигнала для %s", symbol)
		return
	}

	midPrice, err := utils.СalculateWeightedMidPrice(orderBook, 5)
	if err != nil {
		log.Printf("Ошибка получения midPrice для %s: %v", symbol, err)
		return
	}
	log.Printf("midPrice для %s: %f", symbol, midPrice)

	currentCandle := klines[len(klines)-1]
	var entryPrice, stopLoss, takeProfit float64
	entryPrice = currentCandle.Close

	if isLong {
		stopLoss = entryPrice * (1 - DefaultStopLossPct)
		takeProfit = entryPrice * (1 + DefaultTakeProfitPct)
		log.Printf("LONG сигнал для %s: Entry=%.2f, StopLoss=%.2f, TakeProfit=%.2f",
			symbol, entryPrice, stopLoss, takeProfit)
	} else {
		stopLoss = entryPrice * (1 + DefaultStopLossPct)
		takeProfit = entryPrice * (1 - DefaultTakeProfitPct)
		log.Printf("SHORT сигнал для %s: Entry=%.2f, StopLoss=%.2f, TakeProfit=%.2f",
			symbol, entryPrice, stopLoss, takeProfit)
	}

	_, buyPriceF, sellPriceF, stopLossBuyF, stopLossSellF, takeProfitBuyF, takeProfitSellF, ok := s.checkAndFormatPrices(
		category, symbol, entryPrice, entryPrice, stopLoss, stopLoss, takeProfit, takeProfit,
	)
	if !ok {
		return
	}

	quantity := s.PriceCalculator.CalculateQuantity(symbol, entryPrice)
	log.Printf("Рассчитанное количество для %s: %v", symbol, quantity)
	if !s.BalanceService.CheckBalance(entryPrice, quantity, entryPrice, quantity) {
		return
	}

	if isLong {
		err := s.Trading.PlaceLimitOrder(symbol, "buy", buyPriceF, quantity, stopLossBuyF, takeProfitBuyF)
		if err != nil {
			log.Printf("Не удалось разместить LONG ордер: %v", err)
			return
		}
	} else if isShort {
		err := s.Trading.PlaceLimitOrder(symbol, "sell", sellPriceF, quantity, stopLossSellF, takeProfitSellF)
		if err != nil {
			log.Printf("Не удалось разместить SHORT ордер: %v", err)
			return
		}
	}
}

// checkAndFormatPrices приводит цены ордеров к торговым лимитам.
func (s *VPAScalping) checkAndFormatPrices(category, symbol string, buyPrice, sellPrice, stopLossBuy, stopLossSell,
	takeProfitBuy, takeProfitSell float64) (model.TradeLimits, float64, float64, float64, float64, float64, float64, bool) {

	tradeLimit, err := s.Bybit.GetTradeLimitsViaInstruments(category, symbol)
	if err != nil {
		log.Printf("Ошибка получения торговых лимитов: %v", err)
		return tradeLimit, 0, 0, 0, 0, 0, 0, false
	}

	formatPrice := func(price float64) float64 {
		return s.Formatter.FormatPrice(tradeLimit, price)
	}

	buyPriceF := formatPrice(buyPrice)
	sellPriceF := formatPrice(sellPrice)
	stopLossBuyF := formatPrice(stopLossBuy)
	stopLossSellF := formatPrice(stopLossSell)
	takeProfitBuyF := formatPrice(takeProfitBuy)
	takeProfitSellF := formatPrice(takeProfitSell)

	return tradeLimit, buyPriceF, sellPriceF, stopLossBuyF, stopLossSellF, takeProfitBuyF, takeProfitSellF, true
}

func (s *VPAScalping) IsTradingTime() bool {
	now := time.Now().UTC()
	hour := now.Hour()

	return (hour >= 7 && hour < 16) || (hour >= 13 && hour < 22)
}
