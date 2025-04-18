package exchange

import (
	"bybit-bot/internal/model"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"log"
	"math"
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

func CalculateOrderPrices(midPrice float64, decision model.FacadeResponse, commission float64) (float64, float64) {
	const spreadPercent = 0.0001
	adjustmentRaw := 0.0001 * (decision.Buy - decision.Sell)
	adjustment := math.Max(math.Min(adjustmentRaw, 0.001), -0.001)

	rawBuyPrice := midPrice * (1 - spreadPercent - adjustment)
	rawSellPrice := midPrice * (1 + spreadPercent + adjustment)

	buyPrice := rawBuyPrice * (1 + commission)
	sellPrice := rawSellPrice * (1 - commission)
	log.Printf("Raw BUY: %.5f, Raw SELL: %.5f, с комиссией: BUY=%.5f, SELL=%.5f", rawBuyPrice, rawSellPrice, buyPrice, sellPrice)
	return buyPrice, sellPrice
}

func CalculateStopLoss(price float64, isBuy bool, stopLossPercent float64) float64 {
	if isBuy {
		return price * (1 - stopLossPercent)
	}
	return price * (1 + stopLossPercent)
}

func CalculateTakeProfit(orderPrice float64, isBuy bool, profitPercent float64) float64 {
	if isBuy {
		return orderPrice * (1 + profitPercent)
	}
	return orderPrice * (1 - profitPercent)
}
