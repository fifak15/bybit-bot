package strategy

import (
	"bybit-bot/internal/client"
	"bybit-bot/internal/model"
	"bybit-bot/internal/service/event"
	"errors"
	"fmt"
	"github.com/markcheno/go-talib"
	"sync"
	"time"
)

type IndicatorCalculator struct {
	Cache          map[string]map[string]interface{}
	Bybit          *client.ByBit // передаем указатель
	cacheMutex     sync.RWMutex
	CacheDuration  time.Duration
	indicatorMutex sync.Mutex
	WSListener     *event.WSListener
}

// EMA - Exponential Moving Average
func (ic *IndicatorCalculator) EMA(symbol string, period int) (float64, error) {
	cacheKey := fmt.Sprintf("ema_%d", period)
	if val, err := ic.getFromCache(symbol, cacheKey); err == nil {
		return val.(float64), nil
	}

	klines, err := ic.getKLines(symbol, period*2)
	if err != nil {
		return 0, err
	}

	closes := make([]float64, len(klines))
	for i, k := range klines {
		closes[i] = k.Close
	}

	emaValues := talib.Ema(closes, period)
	if len(emaValues) == 0 {
		return 0, errors.New("empty EMA result")
	}

	result := emaValues[len(emaValues)-1]
	ic.setCache(symbol, cacheKey, result)
	return result, nil
}

// ATR - Average True Range
func (ic *IndicatorCalculator) ATR(symbol string, period int) (float64, error) {
	cacheKey := fmt.Sprintf("atr_%d", period)
	if val, err := ic.getFromCache(symbol, cacheKey); err == nil {
		return val.(float64), nil
	}

	klines, err := ic.getKLines(symbol, period*3)
	if err != nil {
		return 0, err
	}

	highs := make([]float64, len(klines))
	lows := make([]float64, len(klines))
	closes := make([]float64, len(klines))

	for i, k := range klines {
		highs[i] = k.High
		lows[i] = k.Low
		closes[i] = k.Close
	}

	atrValues := talib.Atr(highs, lows, closes, period)
	if len(atrValues) == 0 {
		return 0, errors.New("empty ATR result")
	}

	result := atrValues[len(atrValues)-1]
	ic.setCache(symbol, cacheKey, result)
	return result, nil
}

// RSI - Relative Strength Index
func (ic *IndicatorCalculator) RSI(symbol string, period int) (float64, error) {
	cacheKey := fmt.Sprintf("rsi_%d", period)
	if val, err := ic.getFromCache(symbol, cacheKey); err == nil {
		return val.(float64), nil
	}

	klines, err := ic.getKLines(symbol, period*2)
	if err != nil {
		return 0, err
	}

	closes := make([]float64, len(klines))
	for i, k := range klines {
		closes[i] = k.Close
	}

	rsiValues := talib.Rsi(closes, period)
	if len(rsiValues) == 0 {
		return 0, errors.New("empty RSI result")
	}

	result := rsiValues[len(rsiValues)-1]
	ic.setCache(symbol, cacheKey, result)
	return result, nil
}

// MACD - Moving Average Convergence Divergence
func (ic *IndicatorCalculator) MACD(symbol string, fastPeriod, slowPeriod, signalPeriod int) (float64, float64, float64, error) {
	cacheKey := fmt.Sprintf("macd_%d_%d_%d", fastPeriod, slowPeriod, signalPeriod)
	if val, err := ic.getFromCache(symbol, cacheKey); err == nil {
		values := val.([]float64)
		return values[0], values[1], values[2], nil
	}

	klines, err := ic.getKLines(symbol, slowPeriod*3)
	if err != nil {
		return 0, 0, 0, err
	}

	closes := make([]float64, len(klines))
	for i, k := range klines {
		closes[i] = k.Close
	}

	macd, signal, hist := talib.Macd(closes, fastPeriod, slowPeriod, signalPeriod)
	if len(macd) == 0 {
		return 0, 0, 0, errors.New("empty MACD result")
	}

	result := []float64{
		macd[len(macd)-1],
		signal[len(signal)-1],
		hist[len(hist)-1],
	}

	ic.setCache(symbol, cacheKey, result)
	return result[0], result[1], result[2], nil
}

// Вспомогательные методы
func (ic *IndicatorCalculator) getKLines(symbol string, limit int) ([]model.KlineData, error) {
	klines, err := ic.Bybit.GetKlines(symbol, "1", uint64(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to get klines: %v", err)
	}

	if len(klines) < limit/2 { // Минимум половина запрошенных данных
		return nil, fmt.Errorf("not enough klines (%d < %d)", len(klines), limit/2)
	}

	return klines, nil
}

func (ic *IndicatorCalculator) getFromCache(symbol, key string) (interface{}, error) {
	ic.cacheMutex.RLock()
	defer ic.cacheMutex.RUnlock()

	if symbolCache, ok := ic.Cache[symbol]; ok {
		if cached, ok := symbolCache[key]; ok {
			if cacheTime, ok := symbolCache[key+"_time"]; ok {
				if time.Since(cacheTime.(time.Time)) < ic.CacheDuration {
					return cached, nil
				}
			}
		}
	}
	return nil, errors.New("not in cache")
}

func (ic *IndicatorCalculator) setCache(symbol, key string, value interface{}) {
	ic.cacheMutex.Lock()
	defer ic.cacheMutex.Unlock()

	if _, ok := ic.Cache[symbol]; !ok {
		ic.Cache[symbol] = make(map[string]interface{})
	}

	ic.Cache[symbol][key] = value
	ic.Cache[symbol][key+"_time"] = time.Now()
}

// ClearCache очищает кеш для конкретного символа
func (ic *IndicatorCalculator) ClearCache(symbol string) {
	ic.cacheMutex.Lock()
	defer ic.cacheMutex.Unlock()
	delete(ic.Cache, symbol)
}
