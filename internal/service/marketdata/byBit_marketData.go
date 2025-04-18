package marketdata

import (
	"bybit-bot/internal/client"
	"bybit-bot/internal/model"
	"bybit-bot/internal/service/event"
	"log"
	"time"
)

type ByBitMarketData struct {
	Client     *client.ByBit
	WSListener *event.WSListener
}

func (m *ByBitMarketData) GetRecentKlines(symbol, interval string, required int) ([]model.KlineData, bool) {
	log.Printf("[Маркет-данные] Запрос свечей для %s, интервал %s, требуется %d свечей", symbol, interval, required)

	// Получаем сырые данные от API
	raw, err := m.Client.GetKlines(symbol, uint64(required+1))
	if err != nil {
		log.Printf("[Маркет-данные] ОШИБКА: не удалось получить свечи для %s: %v", symbol, err)
		return nil, false
	}

	if len(raw) < required+1 {
		log.Printf("[Маркет-данные] ОШИБКА: получено %d свечей для %s, требуется минимум %d",
			len(raw), symbol, required+1)
		return nil, false
	}

	closed := raw[:len(raw)-1]
	bars := closed[len(closed)-required:]

	if len(bars) > 0 {
		firstTime := time.Unix(bars[0].Start, 0).Format("02.01.2006 15:04")
		lastTime := time.Unix(bars[len(bars)-1].Start, 0).Format("02.01.2006 15:04")
		log.Printf("[Маркет-данные] Успешно получены свечи %s: %d шт. (период %s - %s)",
			symbol, len(bars), firstTime, lastTime)
	} else {
		log.Printf("[Маркет-данные] ПРЕДУПРЕЖДЕНИЕ: получен пустой набор свечей для %s", symbol)
	}

	return bars, true
}
