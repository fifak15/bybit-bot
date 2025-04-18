package event

import (
	"bybit-bot/internal/model"
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
)

type WSListener struct {
	wsConn         *websocket.Conn
	orderBookCache map[string]*model.OrderbookData
	KlineCache     map[string][]model.KlineData // Кэшируем срезы свечей
	mu             sync.RWMutex
	done           chan struct{}
}

func NewWSListener(urlStr string, header http.Header) (*WSListener, error) {
	conn, resp, err := websocket.DefaultDialer.Dial(urlStr, header)
	if err != nil {
		log.Printf("Ошибка подключения к WebSocket: %v", err)
		if resp != nil {
			log.Printf("HTTP статус: %s", resp.Status)
		}
		return nil, err
	}
	log.Printf("Подключение к WebSocket установлено: %s", urlStr)
	return &WSListener{
		wsConn:         conn,
		orderBookCache: make(map[string]*model.OrderbookData),
		KlineCache:     make(map[string][]model.KlineData),
		done:           make(chan struct{}),
	}, nil
}

// SubscribeChannels отправляет запрос на подписку сразу на несколько каналов.
func (w *WSListener) SubscribeChannels(channels []string) error {
	subMsg := map[string]interface{}{
		"op":   "subscribe",
		"args": channels,
	}
	msg, err := json.Marshal(subMsg)
	if err != nil {
		return err
	}
	log.Printf("Отправка запроса на подписку: %s", string(msg))
	if err := w.wsConn.WriteMessage(websocket.TextMessage, msg); err != nil {
		log.Printf("Ошибка отправки запроса на подписку: %v", err)
		return err
	}
	return nil
}

// ListenAll запускает прослушивание WebSocket и обновляет кеши, используя топики в качестве ключей.
func (w *WSListener) ListenAll() {
	go func() {
		for {
			_, message, err := w.wsConn.ReadMessage()
			if err != nil {
				log.Printf("Ошибка чтения из WebSocket: %v", err)
				break
			}

			var temp struct {
				Topic string `json:"topic"`
			}
			if err := json.Unmarshal(message, &temp); err != nil {
				log.Printf("Ошибка парсинга topic: %v", err)
				continue
			}
			if temp.Topic == "" {
				log.Printf("Сообщение не содержит topic")
				continue
			}

			w.mu.Lock()
			func() {
				defer w.mu.Unlock()

				if strings.HasPrefix(temp.Topic, "orderbook") {
					var obMsg model.OrderbookMessage
					if err := json.Unmarshal(message, &obMsg); err != nil {
						log.Printf("Ошибка парсинга orderbook сообщения: %v", err)
						return
					}
					if obMsg.Type != "snapshot" {
						return
					}
					w.orderBookCache[temp.Topic] = &obMsg.Data

				} else if strings.HasPrefix(temp.Topic, "kline") {
					var klMsg model.KlineMessage
					if err := json.Unmarshal(message, &klMsg); err != nil {
						log.Printf("Ошибка парсинга kline сообщения: %v", err)
						return
					}
					if len(klMsg.Data) == 0 {
						log.Printf("Kline сообщение не содержит данных")
						return
					}

					incoming := klMsg.Data[0]
					existing := w.KlineCache[temp.Topic]

					if len(existing) > 0 {
						last := existing[len(existing)-1]
						if last.Start == incoming.Start {
							existing[len(existing)-1] = incoming
							w.KlineCache[temp.Topic] = existing
							log.Printf("Обновлена незакрытая свеча (start=%d) для топика %s. Всего свечей: %d", incoming.Start, temp.Topic, len(existing))
							return
						}
					}
					existing = append(existing, incoming)
					w.KlineCache[temp.Topic] = existing
					log.Printf("Добавлена новая свеча (start=%d, confirm=%v) для топика %s. Всего свечей: %d", incoming.Start, incoming.Confirm, temp.Topic, len(existing))
				}
			}()
		}
		close(w.done)
	}()
}

func (w *WSListener) GetOrderbookByTopic(topic string) (*model.OrderbookData, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	data, ok := w.orderBookCache[topic]
	if !ok {
		return nil, false
	}
	sortedData := model.OrderbookData{
		Symbol: data.Symbol,
		Bids:   make([]model.PriceLevel, len(data.Bids)),
		Asks:   make([]model.PriceLevel, len(data.Asks)),
	}
	copy(sortedData.Bids, data.Bids)
	copy(sortedData.Asks, data.Asks)

	sort.SliceStable(sortedData.Bids, func(i, j int) bool {
		return sortedData.Bids[i].Price > sortedData.Bids[j].Price
	})

	sort.SliceStable(sortedData.Asks, func(i, j int) bool {
		return sortedData.Asks[i].Price < sortedData.Asks[j].Price
	})

	return &sortedData, true
}

// GetKlineByTopic возвращает кешированные данные свечей по топику.
/*func (w *WSListener) GetKlineByTopic(topic string) (*model.KlineData, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	data, ok := w.klineCache[topic]
	return data, ok
}*/

func (w *WSListener) GetKlinesByTopic(topic string) ([]model.KlineData, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Теперь возвращаем срез свечей, а не одну свечу
	klines, ok := w.KlineCache[topic]
	return klines, ok
}

// Также можно добавить методы, которые генерируют топик по символу, если это необходимо.

func (w *WSListener) Close() error {
	log.Printf("Закрытие WebSocket-соединения")
	return w.wsConn.Close()
}
