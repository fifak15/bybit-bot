package exchange

import (
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
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
