package strategy

import (
	"bybit-bot/internal/client"
	"bybit-bot/internal/model"
	"bybit-bot/internal/repository"
	"bybit-bot/internal/service/event"
	"bybit-bot/internal/service/exchange"
	"bybit-bot/internal/utils"
	"github.com/markcheno/go-talib"
	"log"
	"strings"
	"time"
)

// VPAScalping реализует стратегию Volume-Price Action Scalping.
type VPAScalping struct {
	OrderRepository  repository.OrderRepository
	WalletRepository *repository.WalletRepository
	Formatter        *utils.Formatter
	BalanceService   *BalanceService
	PriceCalculator  *exchange.PriceCalculator
	Bybit            *client.ByBit
	WSListener       *event.WSListener
	StopLossPercent  float64 // например, 0.5% за порог (0.005)
}

// Параметры стратегии
const (
	VolumeWindow      = 15  // число свечей для расчёта среднего объёма
	VolumeSpikeFactor = 1.5 // коэффициент объёма, определяющий всплеск
	LookbackPeriod    = 5   // число предыдущих свечей для оценки локального минимума/максимума
	RiskRewardRatio   = 2.0 // Тейк-Профит = риск * RiskRewardRatio
)

// Make – основная функция стратегии VPA Scalping.
// symbol — например, "BTCUSDT", category — тип инструмента (например, "linear")
func (s *VPAScalping) Make(symbol, category string) {
	// 1. Если открытых ордеров уже нет, продолжаем.
	openOrders, err := s.Bybit.GetOpenOrders(category, symbol)
	if err != nil {
		log.Printf("Ошибка получения открытых ордеров для %s: %v", symbol, err)
		return
	}
	if openOrders != nil && len(openOrders.Orders) > 0 {
		log.Printf("Для %s уже есть открытые ордера (%d шт.), пропускаем стратегию", symbol, len(openOrders.Orders))
		return
	}

	// 2. Получаем последние свечи.
	// Требуем количество свечей = VolumeWindow + LookbackPeriod,
	// чтобы корректно рассчитать средний объём и оценить локальный экстремум.
	topicKline := "kline.1." + strings.ToUpper(symbol)
	klines, ok := s.getRecentKlines(topicKline, VolumeWindow+LookbackPeriod)
	if !ok {
		log.Printf("Недостаточно данных свечей для %s", symbol)
		return
	}

	// 3. Получаем данные ордербука (для вычисления midPrice, можно использовать как дополнительный фильтр).
	topicOrderbook := "orderbook.50." + strings.ToUpper(symbol)
	orderBook, ok := s.WSListener.GetOrderbookByTopic(topicOrderbook)
	if !ok || len(orderBook.Bids) == 0 || len(orderBook.Asks) == 0 {
		log.Printf("Недостаточно данных ордербука для %s", symbol)
		return
	}

	// 4. Анализируем последние свечи для генерации сигнала.
	isLong := s.checkLongSignal(klines)
	log.Printf("isLong %s", isLong)
	isShort := s.checkShortSignal(klines)
	log.Printf("isShort %s", isShort)
	if !isLong && !isShort {
		log.Printf("Нет сигнала для %s", symbol)
		return
	}

	midPrice := calculateWeightedMidPrice(orderBook, 3)
	log.Printf("midPrice для %s: %f", symbol, midPrice)

	currentCandle := klines[len(klines)-1]
	var entryPrice, stopLoss, takeProfit float64
	if isLong {
		entryPrice = currentCandle.Close

		stopLoss = currentCandle.Low * 0.995
		risk := entryPrice - stopLoss
		takeProfit = entryPrice + risk*RiskRewardRatio
		log.Printf("LONG сигнал для %s: Entry=%.2f, StopLoss=%.2f, TakeProfit=%.2f", symbol, entryPrice, stopLoss, takeProfit)
	} else {
		entryPrice = currentCandle.Close
		stopLoss = currentCandle.High * 1.005
		risk := stopLoss - entryPrice
		takeProfit = entryPrice - risk*RiskRewardRatio
		log.Printf("SHORT сигнал для %s: Entry=%.2f, StopLoss=%.2f, TakeProfit=%.2f", symbol, entryPrice, stopLoss, takeProfit)
	}

	// 7. Форматируем цены по торговым лимитам.
	_, buyPriceF, sellPriceF, stopLossBuyF, stopLossSellF, takeProfitBuyF, takeProfitSellF, ok := s.checkAndFormatPrices(
		category, symbol, entryPrice, entryPrice, stopLoss, stopLoss, takeProfit, takeProfit,
	)
	if !ok {
		return
	}

	// 8. Определяем количество для ордера (на основе выбранной позиции и риск-менеджмента).
	quantity := s.PriceCalculator.CalculateQuantity(symbol, entryPrice, stopLoss)
	log.Printf("Рассчитанное количество для %s: %v", symbol, quantity)
	if !s.BalanceService.CheckBalance(entryPrice, quantity, entryPrice, quantity) {
		return
	}

	// 9. Размещаем ордер в зависимости от сигнала.
	if isLong {
		s.placeOrder(symbol, "buy", buyPriceF, quantity, stopLossBuyF, takeProfitBuyF)
	} else if isShort {
		s.placeOrder(symbol, "sell", sellPriceF, quantity, stopLossSellF, takeProfitSellF)
	}
}

// getRecentKlines получает срез последних свечей по теме.
// Предполагается, что WSListener реализует метод GetRecentKlinesByTopic.
func (s *VPAScalping) getRecentKlines(topic string, required int) ([]model.KlineData, bool) {
	// Из topic "kline.1.BTCUSDT" достаём symbol = "BTCUSDT"
	symbol := strings.TrimPrefix(topic, "kline.1.")

	raw, err := s.Bybit.GetKlines(symbol, uint64(required+1))
	if err != nil {
		log.Printf("REST kline fetch error for %s: %v", symbol, err)
		return nil, false
	}
	if len(raw) < required+1 {
		log.Printf("REST kline returned %d for %s, need %d+1", len(raw), symbol, required)
		return nil, false
	}

	closed := raw[:len(raw)-1]

	bars := closed[len(closed)-required:]
	return bars, true
}

// checkLongSignal анализирует последние свечи и возвращает true, если выполнены условия для входа в длинную позицию.
func (s *VPAScalping) checkLongSignal(klines []model.KlineData) bool {
	n := len(klines)
	if n < VolumeWindow+LookbackPeriod {
		return false
	}

	current := klines[n-1]

	avgVolume := s.calculateAverageVolume(klines[n-VolumeWindow-1 : n-1])
	if current.Volume < avgVolume*VolumeSpikeFactor {
		return false
	}

	if !s.isLocalLowest(current.Low, klines[n-LookbackPeriod-1:n-1]) {
		return false
	}

	if current.Close <= current.Open {
		return false
	}
	closes := getClosingPrices(klines[n-20 : n-1])
	sma20Results := talib.Sma(closes, 20)
	if len(sma20Results) == 0 {
		return false
	}
	sma20 := sma20Results[len(sma20Results)-1]
	if current.Close <= sma20 {
		return false
	}

	return true
}
func getClosingPrices(klines []model.KlineData) []float64 {
	closes := make([]float64, len(klines))
	for i, k := range klines {
		closes[i] = k.Close
	}
	return closes
}
func (s *VPAScalping) isLocalLowest(currentLow float64, previousKlines []model.KlineData) bool {
	count := 0
	for _, k := range previousKlines {
		if currentLow < k.Low {
			count++
		}
	}

	return count >= len(previousKlines)*2/3
}

func (s *VPAScalping) calculateAverageVolume(klines []model.KlineData) float64 {
	total := 0.0
	for _, k := range klines {
		total += k.Volume
	}
	return total / float64(len(klines))
}

// checkShortSignal анализирует последние свечи и возвращает true, если выполнены условия для входа в короткую позицию.
func (s *VPAScalping) checkShortSignal(klines []model.KlineData) bool {
	n := len(klines)
	if n < VolumeWindow+1 {
		return false
	}
	current := klines[n-1]

	totalVolume := 0.0
	for i := n - VolumeWindow; i < n-1; i++ {
		totalVolume += klines[i].Volume
	}
	avgVolume := totalVolume / float64(VolumeWindow)
	if current.Volume < avgVolume*VolumeSpikeFactor {
		return false
	}

	count := 0
	start := n - LookbackPeriod - 1
	if start < 0 {
		start = 0
	}
	for i := start; i < n-1; i++ {
		if current.High > klines[i].High {
			count++
		}
	}
	if count < LookbackPeriod {
		return false
	}

	if current.Close >= current.Open {
		return false
	}

	return true
}

// calculateWeightedMidPrice вычисляет средневзвешенную цену по ордербуку.
func calculateWeightedMidPrice(orderBook *model.OrderbookData, levels int) float64 {
	if levels <= 0 || len(orderBook.Bids) == 0 || len(orderBook.Asks) == 0 {
		return 0.0
	}

	totalBidValue, totalBidQty := 0.0, 0.0
	nBids := levels
	if len(orderBook.Bids) < levels {
		nBids = len(orderBook.Bids)
	}
	for i := 0; i < nBids; i++ {
		bid := orderBook.Bids[i]
		totalBidValue += bid.Price * bid.Size
		totalBidQty += bid.Size
	}
	weightedBid := 0.0
	if totalBidQty > 0 {
		weightedBid = totalBidValue / totalBidQty
	}

	totalAskValue, totalAskQty := 0.0, 0.0
	nAsks := levels
	if len(orderBook.Asks) < levels {
		nAsks = len(orderBook.Asks)
	}
	for i := 0; i < nAsks; i++ {
		ask := orderBook.Asks[i]
		totalAskValue += ask.Price * ask.Size
		totalAskQty += ask.Size
	}
	weightedAsk := 0.0
	if totalAskQty > 0 {
		weightedAsk = totalAskValue / totalAskQty
	}

	return (weightedBid + weightedAsk) / 2
}

// checkAndFormatPrices приводит цены ордеров к торговым лимитам.
func (s *VPAScalping) checkAndFormatPrices(
	category, symbol string,
	buyPrice, sellPrice, stopLossBuy, stopLossSell, takeProfitBuy, takeProfitSell float64,
) (model.TradeLimits, float64, float64, float64, float64, float64, float64, bool) {

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

// placeOrder размещает ордер через Bybit API и сохраняет запись в БД.
func (s *VPAScalping) placeOrder(symbol, side string, price, quantity, stopLoss, takeProfit float64) {
	resp, err := s.Bybit.CreateOrderViaPlaceOrderFuture(symbol, side, "limit", price, quantity, stopLoss, takeProfit)
	if err != nil {
		log.Printf("Ошибка размещения %s ордера для %s: %v, с ценой %f", side, symbol, err, price)
		return
	}
	log.Printf("%s ордер успешно размещен для %s: %+v", strings.ToUpper(side), symbol, resp)

	orderRecord := &model.Order{
		OrderID:   resp.OrderID,
		Symbol:    symbol,
		Side:      side,
		OrderType: "limit",
		Price:     price,
		Quantity:  quantity,
		StopLoss:  stopLoss,
		Status:    "open",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.OrderRepository.InsertOrder(orderRecord); err != nil {
		log.Printf("Ошибка сохранения ордера в БД: %v", err)
	}
}
