package strategy

import (
	"bybit-bot/internal/model"
	"github.com/markcheno/go-talib"
	"log"
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
	minRequired := sd.volumeWindow + sd.lookbackPeriod
	if n < minRequired {
		log.Printf("[Сигнал] Недостаточно данных для LONG. Имеется: %d, требуется: %d", n, minRequired)
		return false
	}

	current := klines[n-1]
	log.Printf("[Сигнал] Анализ LONG для %s: Открытие=%.2f Макс=%.2f Мин=%.2f Закрытие=%.2f Объем=%.2f",
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

	if !sd.isLocalLowest(current.Low, klines[n-sd.lookbackPeriod-1:n-1]) {
		log.Printf("[Сигнал] Не является локальным минимумом (мин=%.2f)", current.Low)
		return false
	}
	log.Printf("[Сигнал] Локальный минимум подтвержден (мин=%.2f)", current.Low)

	if current.Close <= current.Open {
		log.Printf("[Сигнал] Не бычья свеча (открытие=%.2f, закрытие=%.2f)",
			current.Open, current.Close)
		return false
	}
	log.Printf("[Сигнал] Бычья свеча подтверждена (открытие=%.2f, закрытие=%.2f)",
		current.Open, current.Close)

	if n < 20 {
		log.Printf("[Сигнал] Недостаточно данных для SMA20: имеем %d, нужно 20", n)
		return false
	}

	last20 := klines[n-20 : n]
	closes := sd.getClosingPrices(last20)
	log.Printf("[Сигнал] closes: %v", closes)

	sma20Results := talib.Sma(closes, 20)
	if len(sma20Results) == 0 {
		log.Printf("[Сигнал] Ошибка расчета SMA20")
		return false
	}
	sma20 := sma20Results[len(sma20Results)-1]
	log.Printf("[Сигнал] sma20: %.2f", sma20)

	if current.Close <= sma20 {
		log.Printf("[Сигнал] Цена ниже SMA20 (цена=%.2f, sma20=%.2f)",
			current.Close, sma20)
		return false
	}
	log.Printf("[Сигнал] Цена выше SMA20 (цена=%.2f, sma20=%.2f)",
		current.Close, sma20)

	log.Printf("[Сигнал] СИЛЬНЫЙ СИГНАЛ НА ПОКУПКУ")
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
