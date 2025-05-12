package strategy

import (
	"bybit-bot/internal/model"
	"github.com/markcheno/go-talib"
	"log"
	"math"
)

type SignalDetector struct {
	volumeWindow      int
	volumeSpikeFactor float64
	lookbackPeriod    int
}

func NewSignalDetector() *SignalDetector {
	return &SignalDetector{
		volumeWindow:      15,
		volumeSpikeFactor: 1.5,
		lookbackPeriod:    5,
	}
}

func (sd *SignalDetector) CheckLongSignal(klines []model.KlineData) bool {
	n := len(klines)
	if n < 200 {
		return false
	}

	closes := make([]float64, n)
	for i := range klines {
		closes[i] = klines[i].Close
	}

	ema50 := talib.Ema(closes, 50)
	ema200 := talib.Ema(closes, 200)
	rsi := talib.Rsi(closes, 14)

	latest := n - 1
	current := klines[latest]
	price := current.Close

	if ema50[latest] <= ema200[latest] {
		log.Printf("[Сигнал] EMA50 <= EMA200 (%.2f <= %.2f) — тренд не подтвержден", ema50[latest], ema200[latest])
		return false
	}

	sd.lookbackPeriod = 7
	emaDiff := math.Abs(price - ema50[latest])
	if price > ema50[latest] || emaDiff > ema50[latest]*0.015 {
		log.Printf("[Сигнал] Цена не на адекватном откате к EMA50 (%.2f > %.2f)", price, ema50[latest])
		return false
	}

	if rsi[latest] > rsi[latest-1] && rsi[latest] < 50 {
		log.Printf("[Сигнал] RSI вне диапазона: %.2f", rsi[latest])
		return false
	}

	if !sd.isLocalLowest(current.Low, klines[n-sd.lookbackPeriod-1:n-1]) {
		log.Printf("[Сигнал] Не локальный минимум: %.2f", current.Low)
		return false
	}
	log.Printf("[Сигнал] Локальный минимум подтвержден: %.2f", current.Low)

	if current.Close <= current.Open {
		log.Printf("[Сигнал] Не бычья свеча: open=%.2f, close=%.2f", current.Open, current.Close)
		return false
	}
	log.Printf("[Сигнал] Бычья свеча подтверждена: open=%.2f, close=%.2f", current.Open, current.Close)

	log.Printf("[Сигнал] ✅ LONG сигнал подтвержден")
	return true
}

func (sd *SignalDetector) CheckShortSignal(klines []model.KlineData) bool {
	n := len(klines)
	if n < sd.volumeWindow+1 {
		log.Printf("[Сигнал] Недостаточно данных для SHORT. Имеется: %d, требуется: %d",
			n, sd.volumeWindow+1)
		return false
	}

	current := klines[n-1]
	log.Printf("[Сигнал] Анализ SHORT для %s: Открытие=%.2f Макс=%.2f Мин=%.2f Закрытие=%.2f Объем=%.2f",
		current.Symbol, current.Open, current.High, current.Low, current.Close, current.Volume)

	avgVolume := sd.calculateAverageVolume(klines[n-sd.volumeWindow-1 : n-1])
	volumeRatio := current.Volume / avgVolume

	if volumeRatio < sd.volumeSpikeFactor {
		log.Printf("[Сигнал] Объем недостаточен: %.2f < %.2f (требуется x%.1f)",
			current.Volume, avgVolume, sd.volumeSpikeFactor)
		return false
	}
	log.Printf("[Сигнал] Объем ОК: %.2f > %.2f (x%.1f)",
		current.Volume, avgVolume, volumeRatio)

	if !sd.isLocalHighest(current.High, klines[n-sd.lookbackPeriod-1:n-1]) {
		log.Printf("[Сигнал] Не является локальным максимумом (макс=%.2f)", current.High)
		return false
	}
	log.Printf("[Сигнал] Локальный максимум подтвержден (макс=%.2f)", current.High)

	if current.Close >= current.Open {
		log.Printf("[Сигнал] Не медвежья свеча (открытие=%.2f, закрытие=%.2f)",
			current.Open, current.Close)
		return false
	}
	log.Printf("[Сигнал] Медвежья свеча подтверждена (открытие=%.2f, закрытие=%.2f)",
		current.Open, current.Close)

	closes := sd.getClosingPrices(klines[n-50 : n])
	sma20 := talib.Sma(closes[30:], 20)
	sma50 := talib.Sma(closes, 50)

	if len(sma20) == 0 || len(sma50) == 0 {
		log.Printf("[Сигнал] Ошибка расчета SMA20/SMA50")
		return false
	}
	lastSma20 := sma20[len(sma20)-1]
	lastSma50 := sma50[len(sma50)-1]

	if !(current.Close < lastSma20 && lastSma20 < lastSma50) {
		log.Printf("[Сигнал] Тренд не подтверждён (Close=%.2f, SMA20=%.2f, SMA50=%.2f)",
			current.Close, lastSma20, lastSma50)
		return false
	}
	log.Printf("[Сигнал] Тренд подтвержден (Close=%.2f < SMA20=%.2f < SMA50=%.2f)",
		current.Close, lastSma20, lastSma50)

	atr := sd.getATR(klines[n-20:], 14)
	if atr < 0.5 {
		log.Printf("[Сигнал] Низкая волатильность (ATR=%.2f)", atr)
		return false
	}
	log.Printf("[Сигнал] ATR подтверждён: %.2f", atr)

	log.Printf("[Сигнал] СИЛЬНЫЙ СИГНАЛ НА ПРОДАЖУ")
	return true
}

func (sd *SignalDetector) isLocalLowest(currentLow float64, previousKlines []model.KlineData) bool {
	count := 0
	for _, k := range previousKlines {
		if currentLow < k.Low {
			count++
		}
	}
	return count >= len(previousKlines)*2/3
}

func (sd *SignalDetector) isLocalHighest(currentHigh float64, previousKlines []model.KlineData) bool {
	count := 0
	for _, k := range previousKlines {
		if currentHigh > k.High {
			count++
		}
	}
	return count >= len(previousKlines)*2/3
}

func (sd *SignalDetector) calculateAverageVolume(klines []model.KlineData) float64 {
	total := 0.0
	for _, k := range klines {
		total += k.Volume
	}
	return total / float64(len(klines))
}

func (sd *SignalDetector) getClosingPrices(kl []model.KlineData) []float64 {
	out := make([]float64, len(kl))
	for i, k := range kl {
		out[i] = k.Close
	}
	return out
}

func (sd *SignalDetector) getATR(klines []model.KlineData, period int) float64 {
	highs := make([]float64, len(klines))
	lows := make([]float64, len(klines))
	closes := make([]float64, len(klines))
	for i, k := range klines {
		highs[i] = k.High
		lows[i] = k.Low
		closes[i] = k.Close
	}
	atr := talib.Atr(highs, lows, closes, period)
	if len(atr) == 0 {
		return 0
	}
	return atr[len(atr)-1]
}
