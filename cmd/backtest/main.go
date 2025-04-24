// backtest.go
// Независимый бэктестер, использующий MarketDataService для подгрузки истории
package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	_ "bybit-bot/internal/model"
	"bybit-bot/internal/service"
	"github.com/markcheno/go-talib"
)

// Параметры риск-менеджмента
const (
	atrPeriod              = 9
	stopLossATRMul         = 1.0
	riskRewardRatio        = 2.0
	minSLPercent           = 0.001
	maxSLPercent           = 0.01
	spreadAdjustmentFactor = 0.0005
)

// KlineData представляет один бар OHLCV
type KlineData struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

// Position хранит данные по одной сделке
type Position struct {
	Side       string // "long" или "short"
	EntryPrice float64
	SL         float64
	TP         float64
	OpenIdx    int
	ExitPrice  float64
	ExitIdx    int
}

func main() {

	svc := service.NewMarketDataService()
	// получаем последние 1000 часов для BTCUSDT
	klDto, err := svc.GetHistoricalData("BTCUSDT", "1", 29, "linear")
	if err != nil {
		log.Fatalf("Ошибка получения исторических данных: %v", err)
	}
	// конвертируем в KlineData
	var klines []KlineData
	for i, k := range klDto {
		ts, err := time.Parse("2006-01-02 15:04:05", k.StartTime)
		if err != nil {
			log.Fatalf("Неверный формат времени в элементе %d: %v", i, err)
		}
		o, _ := strconv.ParseFloat(k.OpenPrice, 64)
		h, _ := strconv.ParseFloat(k.HighPrice, 64)
		l, _ := strconv.ParseFloat(k.LowPrice, 64)
		c, _ := strconv.ParseFloat(k.ClosePrice, 64)
		v, _ := strconv.ParseFloat(k.Volume, 64)

		klines = append(klines, KlineData{
			Time:   ts,
			Open:   o,
			High:   h,
			Low:    l,
			Close:  c,
			Volume: v,
		})
	}

	positions := Backtest(klines)
	analyze(positions)
}

func Backtest(klines []KlineData) []Position {
	var positions []Position
	n := len(klines)
	for i := atrPeriod + 1; i < n-1; i++ {
		entryIdx := i + 1
		entry := klines[entryIdx].Open
		for _, side := range []string{"long", "short"} {
			sl, tp := CalculateSLTP(side, entry, klines[:entryIdx])
			pos := Position{Side: side, EntryPrice: entry, SL: sl, TP: tp, OpenIdx: entryIdx}
			// симуляция выхода
			for j := entryIdx + 1; j < n; j++ {
				h, l := klines[j].High, klines[j].Low
				if side == "long" {
					if l <= sl {
						pos.ExitPrice = sl
						pos.ExitIdx = j
						break
					}
					if h >= tp {
						pos.ExitPrice = tp
						pos.ExitIdx = j
						break
					}
				} else {
					if h >= sl {
						pos.ExitPrice = sl
						pos.ExitIdx = j
						break
					}
					if l <= tp {
						pos.ExitPrice = tp
						pos.ExitIdx = j
						break
					}
				}
				if j == n-1 && pos.ExitIdx == 0 {
					pos.ExitPrice = klines[j].Close
					pos.ExitIdx = j
				}
			}
			positions = append(positions, pos)
		}
	}
	return positions
}

// analyze выводит статистику по результатам
func analyze(positions []Position) {
	var totalPnL float64
	wins, losses := 0, 0

	for _, p := range positions {
		var pnl float64
		if p.Side == "long" {
			pnl = (p.ExitPrice - p.EntryPrice) / p.EntryPrice
		} else {
			pnl = (p.EntryPrice - p.ExitPrice) / p.EntryPrice
		}
		totalPnL += pnl
		if pnl > 0 {
			wins++
		} else {
			losses++
		}
	}

	fmt.Printf("Trades: %d, Wins: %d, Losses: %d, Total PnL: %.2f%%\n",
		len(positions), wins, losses, totalPnL*100)
}

// CalculateSLTP рассчитывает уровни StopLoss и TakeProfit
func CalculateSLTP(side string, entry float64, klines []KlineData) (sl, tp float64) {
	n := len(klines)
	if n < atrPeriod+1 {
		sl = entry * (1 - minSLPercent)
		tp = entry * (1 + minSLPercent*riskRewardRatio)
		return
	}

	hArr := make([]float64, atrPeriod+1)
	lArr := make([]float64, atrPeriod+1)
	cArr := make([]float64, atrPeriod+1)
	for i := 0; i <= atrPeriod; i++ {
		bar := klines[n-atrPeriod-1+i]
		hArr[i], lArr[i], cArr[i] = bar.High, bar.Low, bar.Close
	}
	atrArr := talib.Atr(hArr, lArr, cArr, atrPeriod)
	atr := atrArr[len(atrArr)-1]

	raw := atr * stopLossATRMul
	minD := entry * minSLPercent
	maxD := entry * maxSLPercent
	d := clamp(raw, minD, maxD) + entry*spreadAdjustmentFactor

	switch side {
	case "long":
		sl = entry - d
		tp = entry + d*riskRewardRatio
	case "short":
		sl = entry + d
		tp = entry - d*riskRewardRatio
	default:
		sl, tp = entry, entry
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
