package main

import (
	"bybit-bot/internal/backtest"
	"bybit-bot/internal/service/strategy"
	"log"
	_ "time"
)

func main() {
	err := backtest.DownLoadKlinesFromBybit()
	if err != nil {
		log.Fatalf("Ошибка при загрузке данных: %v", err)
	}
	klines, err := backtest.LoadKlinesFromCSV("bybit_klines.csv")
	if err != nil {
		log.Fatalf("Ошибка загрузки CSV: %v", err)
	}

	sd := &strategy.SignalDetector{}
	result := backtest.RunBacktest(klines, sd.CheckLongSignal, sd.CheckShortSignal)

	log.Printf("Сделок: %d | Побед: %d | Поражений: %d | WinRate: %.2f%% | PnL: %.4f",
		result.NumTrades, result.NumWins, result.NumLosses, result.WinRate, result.TotalPnL)
}
