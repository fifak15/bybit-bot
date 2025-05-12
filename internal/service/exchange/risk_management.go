package exchange

import (
	"bybit-bot/internal/model"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"log"
)

const (
	atrPeriod       = 11
	stopLossATRMul  = 1.0
	riskRewardRatio = 2.0
	minSLPercent    = 0.001
	maxSLPercent    = 0.01

	takerFee = 0.0010  // 0.10 %
	makerFee = 0.00036 // 0.036 %
)

func RiskManagement(stopLoss, takeProfit float64) order.RiskManagementModes {
	rm := order.RiskManagementModes{Mode: "Full"}
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

// CalculateSLTP рассчитывает скорректированные SL и TP с учётом комиссий
func CalculateSLTP(side string, entry float64, klines []model.KlineData) (sl, tp float64) {
	const (
		slPercent = 0.006 // 0.6%
		tpPercent = 0.015 // 1.5%
	)

	var entryNet float64
	if side == "long" {
		entryNet = entry * (1 + takerFee)
	} else {
		entryNet = entry * (1 - takerFee)
	}

	switch side {
	case "long":
		sl = entryNet * (1 - slPercent) / (1 - makerFee)
		tp = entryNet * (1 + tpPercent) / (1 - makerFee)
		log.Printf("[LONG] entry=%.8f, entryNet=%.8f, SL=%.8f, TP=%.8f", entry, entryNet, sl, tp)

	case "short":
		sl = entryNet * (1 + slPercent) / (1 - makerFee)
		tp = entryNet * (1 - tpPercent) / (1 - makerFee)
		log.Printf("[SHORT] entry=%.8f, entryNet=%.8f, SL=%.8f, TP=%.8f", entry, entryNet, sl, tp)

	default:
		sl, tp = entry, entry
		log.Printf("[RM] Неизвестный side='%s'", side)
	}

	return
}

func clamp(x, min, max float64) float64 {
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	return x
}
