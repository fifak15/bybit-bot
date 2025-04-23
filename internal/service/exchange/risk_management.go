package exchange

import (
	"bybit-bot/internal/model"
	"github.com/markcheno/go-talib"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	ATRPeriod             = 14
	StopLossATRMultiplier = 1.0 // во сколько ATR задаём стоп-лосс
	RiskRewardRatio       = 2.0 // тейк-профит = distance * R
)

// RiskManagement формирует RiskManagementModes на основе переданных параметров стоп‑лосса и тейк‑профита.
// Если значение stopLoss или takeProfit больше 0, соответствующий режим включается.
func RiskManagement(stopLoss, takeProfit float64) order.RiskManagementModes {
	rm := order.RiskManagementModes{
		Mode: "Full",
	}
	if takeProfit > 0 {
		rm.TakeProfit = order.RiskManagement{
			Enabled:          true,
			TriggerPriceType: order.LastPrice,
			Price:            takeProfit,
			OrderType:        order.Limit,
		}
	}
	if stopLoss > 0 {
		rm.StopLoss = order.RiskManagement{
			Enabled:          true,
			TriggerPriceType: order.LastPrice,
			Price:            stopLoss,
			LimitPrice:       stopLoss,
			OrderType:        order.Limit,
		}
	}
	return rm
}

// CalculateSLTP даёт динамический стоп-лосс и тейк-профит для лонга/шорта.
// side = "long" или "short".
func CalculateSLTP(side string, entryPrice float64, klines []model.KlineData) (stopLoss, takeProfit float64) {
	n := len(klines)
	if n < ATRPeriod+1 {
		return entryPrice * 0.995, entryPrice * 1.005
	}

	highs := make([]float64, ATRPeriod+1)
	lows := make([]float64, ATRPeriod+1)
	closes := make([]float64, ATRPeriod+1)
	for i := 0; i <= ATRPeriod; i++ {
		k := klines[n-ATRPeriod-1+i]
		highs[i] = k.High
		lows[i] = k.Low
		closes[i] = k.Close
	}
	atrArr := talib.Atr(highs, lows, closes, ATRPeriod)
	atr := atrArr[len(atrArr)-1]

	distance := atr * StopLossATRMultiplier
	if side == "long" {
		stopLoss = entryPrice - distance
		takeProfit = entryPrice + distance*RiskRewardRatio
	} else {
		stopLoss = entryPrice + distance
		takeProfit = entryPrice - distance*RiskRewardRatio
	}
	return
}
