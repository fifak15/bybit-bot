package backtest

import (
	"bybit-bot/internal/model"
	"bybit-bot/internal/service/exchange"
)

type Trade struct {
	Side       string
	EntryTime  model.KlineData
	ExitTime   model.KlineData
	EntryPrice float64
	ExitPrice  float64
	Result     string
	Profit     float64
}

type BacktestResult struct {
	Trades    []Trade
	TotalPnL  float64
	WinRate   float64
	NumWins   int
	NumLosses int
	NumTrades int
}

func RunBacktest(
	klines []model.KlineData,
	longSignal func([]model.KlineData) bool,
	shortSignal func([]model.KlineData) bool,
) BacktestResult {
	const lookback = 120

	var trades []Trade
	for i := lookback; i < len(klines)-1; i++ {
		window := klines[i-lookback : i]
		current := klines[i]

		var side string
		if longSignal(window) {
			side = "long"
		} else if shortSignal(window) {
			side = "short"
		} else {
			continue
		}

		entryPrice := current.Close
		sl, tp := exchange.CalculateSLTP(side, entryPrice, window)

		exitPrice := entryPrice
		exitIndex := i
		result := "none"

		for j := i + 1; j < len(klines); j++ {
			c := klines[j]

			if side == "long" {
				if c.Low <= sl {
					exitPrice = sl
					result = "sl"
					exitIndex = j
					break
				}
				if c.High >= tp {
					exitPrice = tp
					result = "tp"
					exitIndex = j
					break
				}
			} else {
				if c.High >= sl {
					exitPrice = sl
					result = "sl"
					exitIndex = j
					break
				}
				if c.Low <= tp {
					exitPrice = tp
					result = "tp"
					exitIndex = j
					break
				}
			}
		}

		var profit float64
		if side == "long" {
			profit = (exitPrice - entryPrice) / entryPrice
		} else {
			profit = (entryPrice - exitPrice) / entryPrice
		}

		trades = append(trades, Trade{
			Side:       side,
			EntryTime:  current,
			ExitTime:   klines[exitIndex],
			EntryPrice: entryPrice,
			ExitPrice:  exitPrice,
			Result:     result,
			Profit:     profit,
		})

		i = exitIndex
	}

	var totalPnL float64
	var wins, losses int

	for _, t := range trades {
		totalPnL += t.Profit
		if t.Result == "tp" {
			wins++
		} else if t.Result == "sl" {
			losses++
		}
	}

	winRate := 0.0
	totalTrades := len(trades)
	if totalTrades > 0 {
		winRate = float64(wins) / float64(totalTrades) * 100
	}

	return BacktestResult{
		Trades:    trades,
		TotalPnL:  totalPnL,
		WinRate:   winRate,
		NumWins:   wins,
		NumLosses: losses,
		NumTrades: totalTrades,
	}
}
