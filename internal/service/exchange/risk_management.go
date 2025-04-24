package exchange

import (
	"github.com/markcheno/go-talib"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"log"

	"bybit-bot/internal/model"
)

// Параметры риск‑менеджмента
const (
	atrPeriod              = 11
	stopLossATRMul         = 1.0    // сколько ATR в расстоянии до SL
	riskRewardRatio        = 2.0    // коэффициент RRR
	minSLPercent           = 0.001  // 0.1%
	maxSLPercent           = 0.01   // 1%
	spreadAdjustmentFactor = 0.0005 // 0.05%
)

// RiskManagement формирует режимы StopLoss и TakeProfit для ордера
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

func CalculateSLTP(side string, entry float64, klines []model.KlineData) (sl, tp float64) {
	n := len(klines)
	if n < atrPeriod+1 {
		sl = entry * (1 - minSLPercent)
		tp = entry * (1 + minSLPercent*riskRewardRatio)
		log.Printf("[RM] Недостаточно данных для ATR (%d баров), SL=%.8f, TP=%.8f", n, sl, tp)
		return
	}

	// Собираем OHLC для расчёта ATR
	h := make([]float64, atrPeriod+1)
	l := make([]float64, atrPeriod+1)
	c := make([]float64, atrPeriod+1)
	for i := 0; i <= atrPeriod; i++ {
		bar := klines[n-atrPeriod-1+i]
		h[i], l[i], c[i] = bar.High, bar.Low, bar.Close
	}
	atrArr := talib.Atr(h, l, c, atrPeriod)
	atr := atrArr[len(atrArr)-1]
	log.Printf("[RM] ATR(%d)=%.8f", atrPeriod, atr)

	// Расстояние до SL с учётом min/max и спреда
	rawDist := atr * stopLossATRMul
	minDist := entry * minSLPercent
	maxDist := entry * maxSLPercent
	d := clamp(rawDist, minDist, maxDist)
	d += entry * spreadAdjustmentFactor
	log.Printf("[RM] Dist(raw=%.8f,min=%.8f,max=%.8f,spread=%.8f)=%.8f", rawDist, minDist, maxDist, entry*spreadAdjustmentFactor, d)

	switch side {
	case "long":
		sl = entry - d
		tp = entry + d*riskRewardRatio
		log.Printf("[LONG] Entry=%.8f → SL=%.8f, TP=%.8f", entry, sl, tp)

	case "short":
		sl = entry + d
		tp = entry - d*riskRewardRatio
		if tp >= entry {
			tp = entry - d*riskRewardRatio
			log.Printf("[SHORT] TP>=Entry, пересчитан TP=%.8f", tp)
		}
		log.Printf("[SHORT] Entry=%.8f → SL=%.8f, TP=%.8f", entry, sl, tp)

	default:
		sl, tp = entry, entry
		log.Printf("[RM] Неизвестный side='%s', SL=Entry, TP=Entry", side)
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
