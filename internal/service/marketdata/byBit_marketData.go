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
	log.Printf("[Маркет-данные] Запрос %d свечей %s, интервал '%s'", required, symbol, interval)

	raw, err := m.Client.GetKlines(symbol, uint64(required+1))
	if err != nil {
		log.Printf("[Маркет-данные] ОШИБКА запроса: %v", err)
		return nil, false
	}

	for i := range raw {
		raw[i].Start = raw[i].Start / 1000
		raw[i].End = raw[i].End / 1000
	}

	if len(raw) == 0 {
		log.Printf("[Маркет-данные] ПРЕДУПРЕЖДЕНИЕ: пустой ответ от API")
		return nil, false
	}

	firstBar := raw[0]
	now := time.Now().Unix()

	if firstBar.Start < 1609459200 || firstBar.Start > now+86400 {
		log.Printf("[Маркет-данные] ОШИБКА: некорректный timestamp %d (%s)",
			firstBar.Start,
			time.Unix(firstBar.Start, 0).Format("2006-01-02 15:04:05"))
		return nil, false
	}

	closed := raw[:len(raw)-1]
	bars := closed[len(closed)-required:]

	if len(bars) > 0 {
		firstTime := time.Unix(bars[0].Start, 0).Format("2006-01-02 15:04")
		lastTime := time.Unix(bars[len(bars)-1].Start, 0).Format("2006-01-02 15:04")
		log.Printf("[Маркет-данные] Успешно получено %d свечей: %s - %s",
			len(bars), firstTime, lastTime)
	}

	return bars, true
}
