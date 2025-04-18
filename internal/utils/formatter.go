package utils

import (
	"bybit-bot/internal/model"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
)

type Formatter struct {
}

func (m *Formatter) FormatPrice(limit model.TradeLimits, price float64) float64 {
	if price < limit.MinNotional {
		return limit.MinNotional
	}

	split := strings.Split(fmt.Sprintf("%s", strconv.FormatFloat(limit.TickSize, 'f', -1, 64)), ".")
	precision := 0
	if len(split) > 1 {
		precision = len(split[1])
	}
	ratio := math.Pow(10, float64(precision))
	return math.Round(price*ratio) / ratio
}

func (m *Formatter) FormatQuantity(limit model.TradeLimits, quantity float64) float64 {
	if quantity < limit.MinQuantity {
		return limit.MinQuantity
	}

	splitQty := strings.Split(fmt.Sprintf("%s", strconv.FormatFloat(quantity, 'f', -1, 64)), ".")
	split := strings.Split(fmt.Sprintf("%s", strconv.FormatFloat(limit.MinQuantity, 'f', -1, 64)), ".")
	precision := 0
	if len(split) > 1 {
		precision = len(split[1])
	}

	second := "00"
	if precision > 0 && len(splitQty) > 1 {
		substr := precision
		if len(splitQty[1]) < substr {
			substr = len(splitQty[1])
		}

		second = splitQty[1][0:substr]
	}
	quantity, _ = strconv.ParseFloat(fmt.Sprintf("%s.%s", splitQty[0], second), 64)

	return quantity
}

func (m *Formatter) ComparePercentage(first float64, second float64) model.Percent {
	return model.Percent(second * 100.00 / first)
}

func (m *Formatter) Round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func (m *Formatter) ToFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(m.Round(num*output)) / output
}

func (m *Formatter) Floor(num float64) int64 {
	return int64(math.Floor(num))
}

func (m *Formatter) ByBitSideToBinanceSide(side string) string {
	switch side {
	case "Sell":
		return "SELL"
	case "Buy":
		return "BUY"
	default:
		log.Panicf("Side %s is not supported by ByBitSideToBinanceSide", side)
	}

	return ""
}

func (m *Formatter) ByBitTypeToBinanceType(orderType string) string {
	switch orderType {
	case "Limit":
		return "LIMIT"
	default:
		log.Panicf("Order type %s is not supported by ByBitTypeToBinanceType", orderType)
	}

	return ""
}

func (m *Formatter) ByBitSymbolStatusToBinanceSymbolStatus(status string) string {
	switch status {
	case "Trading":
		return "TRADING"
	}

	return ""
}
