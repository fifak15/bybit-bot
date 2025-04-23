package client

import (
	"bybit-bot/internal/model"
	"bybit-bot/internal/service/exchange"
	"context"
	"errors"
	"fmt"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchanges "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bybit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ByBit представляет обёртку для работы с клиентом Bybit.
// Поле client хранит указатель на клиента библиотеки GoCryptoTrader,
// инициализированного с необходимыми настройками (API-ключ, секрет и т.д.).
// cachedTradeLimits хранит полученные торговые лимиты и время их получения.
type cachedTradeLimits struct {
	limits    model.TradeLimits
	timestamp time.Time
}

type ByBit struct {
	APIKey           string
	APISecret        string
	client           *bybit.Bybit
	tradeLimitsCache map[string]cachedTradeLimits
	cacheMu          sync.RWMutex
}

func NewByBit(apiKey, apiSecret string) *ByBit {
	client := &bybit.Bybit{}
	client.SetDefaults()

	client.API.Endpoints = client.NewEndpoints()
	err := client.API.Endpoints.SetDefaultEndpoints(map[exchanges.URL]string{
		exchanges.RestFutures:      "https://api-demo.bybit.com",
		exchanges.RestUSDTMargined: "https://api-demo.bybit.com",
		exchanges.RestSpot:         "https://api-demo.bybit.com",
	})
	if err != nil {
		log.Printf("Ошибка установки конечных точек: %v", err)

		return nil
	}

	client.SetCredentials(apiKey, apiSecret, "", "", "", "")
	log.Printf("Bybit клиент инициализирован с API ключом: %s", apiKey)

	return &ByBit{
		APIKey:           apiKey,
		APISecret:        apiSecret,
		client:           client,
		tradeLimitsCache: make(map[string]cachedTradeLimits),
	}
}

func (b *ByBit) GetKlines(symbol, intervalStr string, limit uint64) ([]model.KlineData, error) {
	ctx := context.Background()
	category := "linear"
	var intervalEnum kline.Interval
	var intervalMinutes int
	switch intervalStr {
	case "1":
		intervalEnum = kline.OneMin
		intervalMinutes = 1
	case "5":
		intervalEnum = kline.FiveMin
		intervalMinutes = 5
	default:
		return nil, fmt.Errorf("unsupported interval: %s", intervalStr)
	}

	startTime := time.Now().Add(-time.Duration(limit*uint64(intervalMinutes)) * time.Minute)
	endTime := time.Now()

	raw, err := b.client.GetKlines(ctx, category, symbol, intervalEnum, startTime, endTime, limit)
	if err != nil {
		return nil, fmt.Errorf("error getting Klines: %w", err)
	}

	var out []model.KlineData
	for i, it := range raw {
		startMs := it.StartTime.UnixNano() / int64(time.Millisecond)
		endMs := startMs + int64(time.Minute/time.Millisecond) - 1

		confirm := true
		if i == len(raw)-1 {
			confirm = false
		}

		out = append(out, model.KlineData{
			Start:     startMs,
			End:       endMs,
			Interval:  strconv.Itoa(int(intervalEnum)),
			Open:      it.Open,
			High:      it.High,
			Low:       it.Low,
			Close:     it.Close,
			Volume:    it.TradeVolume,
			Turnover:  it.Turnover,
			Confirm:   confirm,
			Timestamp: startMs,
		})
	}

	return out, nil
}

// GetDepth возвращает данные ордербука для заданного символа и лимита.
func (b *ByBit) GetDepth(symbol, category string, limit int64) *model.OrderbookData {
	ctx := context.Background()
	if limit > 200 {
		limit = 200
	}

	orderBook, err := b.client.GetOrderBook(ctx, category, symbol, limit)
	if err != nil {
		log.Fatalf("Error getting OrderBook: %s", err)
	}

	modelOrderBook := &model.OrderbookData{
		Bids: make([]model.PriceLevel, len(orderBook.Bids)),
		Asks: make([]model.PriceLevel, len(orderBook.Asks)),
	}

	for i, bid := range orderBook.Bids {
		modelOrderBook.Bids[i] = model.PriceLevel{
			Price: bid.Price,
			Size:  bid.Amount,
		}
	}
	for i, ask := range orderBook.Asks {
		modelOrderBook.Asks[i] = model.PriceLevel{
			Price: ask.Price,
			Size:  ask.Amount,
		}
	}
	return modelOrderBook
}

// GetTickers возвращает данные тикеров для заданных символов.
func (b *ByBit) GetTickers(category, symbol string) []model.WSTickerPrice {
	ctx := context.Background()
	tickers := make([]model.WSTickerPrice, 0)
	tickerData, err := b.client.GetTickers(ctx, category, symbol, "", time.Time{})
	if err != nil {
		log.Fatalf("Ошибка при получении тикеров: %s", err)
	}

	for _, t := range tickerData.List {
		if symbol == "" || t.Symbol == symbol {
			tickers = append(tickers, model.WSTickerPrice{
				Symbol: t.Symbol,
				Price:  t.LastPrice.Float64(),
			})
		}
	}
	return tickers
}

func (b *ByBit) GetTradingRequirements(category, symbol string) (*model.TradingRequirements, error) {
	ctx := context.Background()

	req, err := b.client.GetFeeRate(ctx, category, symbol, "")
	log.Printf("req!!: %v", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get trading requirements: %w", err)
	}

	// Ищем в списке комиссий запись для нужного символа (без учета регистра)
	var makerFee, takerFee types.Number
	found := false
	for _, fee := range req.List {
		if strings.ToUpper(fee.Symbol) == strings.ToUpper(symbol) {
			makerFee = fee.Maker
			takerFee = fee.Taker
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("fee data for symbol %s not found", symbol)
	}

	tradingReq := &model.TradingRequirements{
		MakerFee: makerFee,
		TakerFee: takerFee,
	}
	log.Printf("Получены торговые требования для %s: MakerFee=%v, TakerFee=%v", symbol, tradingReq.MakerFee, tradingReq.TakerFee)
	return tradingReq, nil
}

func (b *ByBit) GetTradingFees(category, symbol string) (float64, error) {

	tradingReq, err := b.GetTradingRequirements(category, symbol)
	if err != nil {
		return 0, fmt.Errorf("failed to get trading requirements: %w", err)
	}

	takerFee := tradingReq.TakerFee
	return float64(takerFee), nil
}

// GetOpenOrders смотрит открытые ордера
func (b *ByBit) GetOpenOrders(category, symbol string) (*model.TradeOrders, error) {
	ctx := context.Background()
	libOrders, err := b.client.GetOpenOrders(ctx, category, symbol, "", "", "", "", "", "", 0, 10)
	if err != nil {
		return nil, fmt.Errorf("error getting open orders for %s: %w", symbol, err)
	}
	modelOrders := convertTradeOrders(libOrders)
	return modelOrders, nil
}

func convertTradeOrders(libOrders *bybit.TradeOrders) *model.TradeOrders {
	if libOrders == nil {
		return nil
	}
	orders := make([]model.Order, len(libOrders.List))
	for i, o := range libOrders.List {
		orders[i] = model.Order{
			OrderID: o.OrderID,
			Symbol:  o.Symbol,
			Side:    o.Side,
			Price:   float64(o.Price),
		}
	}

	return &model.TradeOrders{
		Orders: orders,
	}
}

// CreateOrder размещает ордер через клиента Bybit.
func (b *ByBit) CreateOrderSpot(symbol, side, orderType string, price, amount, stopLoss, takeProfit float64) (*order.SubmitResponse, error) {
	ctx := context.Background()

	pair, err := currency.NewPairFromString(symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to create currency pair from symbol %s: %v", symbol, err)
	}

	var orderSide order.Side
	switch side {
	case "buy", "Buy":
		orderSide = order.Buy
	case "sell", "Sell":
		orderSide = order.Sell
	default:
		return nil, fmt.Errorf("invalid order side: %s", side)
	}

	var orderTypeVal order.Type
	switch orderType {
	case "limit", "Limit":
		orderTypeVal = order.Limit
	case "market", "Market":
		orderTypeVal = order.Market
	default:
		return nil, fmt.Errorf("unsupported order type: %s", orderType)
	}

	submitOrder := &order.Submit{
		Exchange:  "Bybit",
		Pair:      pair,
		Side:      orderSide,
		Type:      orderTypeVal,
		Price:     price,
		Amount:    amount,
		AssetType: asset.Spot,
	}

	// Если заданы стоп‑лосс или тейк‑профит, формируем RiskManagementModes через отдельную функцию.
	if stopLoss > 0 || takeProfit > 0 {
		submitOrder.RiskManagementModes = exchange.RiskManagement(stopLoss, takeProfit)
	}

	if b.client.API.Endpoints != nil {
		log.Printf("Отправка ордера на URL: %v", b.client.API.Endpoints)
	} else {
		log.Printf("Endpoints не настроены")
	}

	response, err := b.client.SubmitOrder(ctx, submitOrder)
	if err != nil {
		log.Printf("Ошибка размещения ордера: %v", err)
		return nil, err
	}
	return response, nil
}

func (b *ByBit) GetTradeLimitsViaInstruments(category, symbol string) (model.TradeLimits, error) {
	cacheKey := category + ":" + symbol
	b.cacheMu.RLock()
	if cached, found := b.tradeLimitsCache[cacheKey]; found {
		if time.Since(cached.timestamp) < 2*time.Hour {
			b.cacheMu.RUnlock()
			log.Printf("Используем кэшированные торговые лимиты для %s", cacheKey)
			return cached.limits, nil
		}
	}
	b.cacheMu.RUnlock()

	ctx := context.Background()
	instruments, err := b.client.GetInstrumentInfo(ctx, category, symbol, "", "", "", 0)
	if err != nil {
		return model.TradeLimits{}, fmt.Errorf("failed to get instrument info: %w", err)
	}
	for _, inst := range instruments.List {
		if inst.Symbol == symbol {
			limits := model.TradeLimits{
				Symbol:      symbol,
				MinQuantity: float64(inst.LotSizeFilter.MinOrderQty),
				MaxQuantity: float64(inst.LotSizeFilter.MaxOrderQty),
				StepSize:    float64(inst.LotSizeFilter.QtyStep),
				TickSize:    float64(inst.PriceFilter.TickSize),
				MinNotional: float64(inst.PriceFilter.MinPrice),
				MaxNotional: float64(inst.PriceFilter.MaxPrice),
			}

			b.cacheMu.Lock()
			b.tradeLimitsCache[cacheKey] = cachedTradeLimits{
				limits:    limits,
				timestamp: time.Now(),
			}
			b.cacheMu.Unlock()

			return limits, nil
		}
	}
	return model.TradeLimits{}, errors.New("instrument not found")
}

// CreateOrder размещает ордер через клиента Bybit.
func (b *ByBit) CreateOrderViaPlaceOrderFuture(symbol, side, orderType string, price, amount, stopLoss, takeProfit float64) (*order.SubmitResponse, error) {
	ctx := context.Background()

	// Создаем валютную пару.
	pair, err := currency.NewPairFromString(symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to create currency pair from symbol %s: %v", symbol, err)
	}

	var orderSide string
	switch side {
	case "buy", "Buy":
		orderSide = "Buy"
	case "sell", "Sell":
		orderSide = "Sell"
	default:
		return nil, fmt.Errorf("invalid order side: %s", side)
	}

	// Определяем тип ордера.
	var orderTypeVal order.Type
	switch strings.ToLower(orderType) {
	case "limit":
		orderTypeVal = order.Limit
	case "market":
		orderTypeVal = order.Market
	default:
		return nil, fmt.Errorf("unsupported order type: %s", orderType)
	}

	arg := &bybit.PlaceOrderParams{
		Category:       getCategoryName(asset.USDTMarginedFutures),
		Symbol:         pair,
		Side:           orderSide,
		OrderType:      orderTypeToString(orderTypeVal),
		OrderQuantity:  amount,
		Price:          price,
		ReduceOnly:     false,
		CloseOnTrigger: false,
		TpslMode:       "Full",
	}

	if stopLoss > 0 || takeProfit > 0 {
		if takeProfit > 0 {
			arg.TakeProfitPrice = takeProfit
			arg.TakeProfitTriggerBy = "MarkPrice"
		}
		if stopLoss > 0 {
			arg.StopLossPrice = stopLoss
			arg.StopLossTriggerBy = "MarkPrice"
			// Для рыночного SL НЕ указываем SlLimitPrice:
			arg.SlOrderType = orderTypeToString(order.Market)
			// arg.SlLimitPrice — удалено
		}
	}
	if b.client.API.Endpoints != nil {
		log.Printf("Отправка ордера на URL: %v", b.client.API.Endpoints)
	} else {
		log.Printf("Endpoints не настроены")
	}

	// Отправляем ордер через метод PlaceOrder.
	response, err := b.client.PlaceOrder(ctx, arg)
	if err != nil {
		log.Printf("Ошибка размещения ордера через PlaceOrder: %v", err)
		return nil, err
	}

	// Преобразуем полученный ответ в структуру SubmitResponse.
	submitResp, err := (&order.Submit{}).DeriveSubmitResponse(response.OrderID)
	if err != nil {
		return nil, err
	}
	submitResp.Status = order.New
	return submitResp, nil
}

func getCategoryName(assetType asset.Item) string {
	switch assetType {
	case asset.USDTMarginedFutures:
		return "linear"
	case asset.Spot:
		return "spot"
	default:
		return ""
	}
}
func orderTypeToString(t order.Type) string {
	switch t {
	case order.Limit:
		return "Limit"
	case order.Market:
		return "Market"
	default:
		return ""
	}
}

func (b *ByBit) GetBalance() (*bybit.WalletBalance, error) {
	ctx := context.Background()
	balance, err := b.client.GetWalletBalance(ctx, "UNIFIED", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet balance: %w", err)
	}
	log.Printf("Получен баланс: %+v", balance)
	return balance, nil
}
