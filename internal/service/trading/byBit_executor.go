package trading

import (
	"bybit-bot/internal/client"
	"bybit-bot/internal/model"
	"bybit-bot/internal/repository"
	"log"
	"time"
)

type ByBitExecutor struct {
	API  *client.ByBit
	Repo repository.OrderRepository
}

func (e *ByBitExecutor) PlaceLimitOrder(symbol, side string, price, qty, stopLoss, takeProfit float64) error {
	resp, err := e.API.CreateOrderViaPlaceOrderFuture(symbol, side, "limit", price, qty, stopLoss, takeProfit)
	if err != nil {
		log.Printf("Ошибка размещения %s ордера для %s: %v, с ценой %f", side, symbol, err, price)
		return nil
	}
	rec := &model.Order{
		OrderID:   resp.OrderID,
		Symbol:    symbol,
		Side:      side,
		OrderType: "limit",
		Price:     price,
		Quantity:  qty,
		StopLoss:  stopLoss,
		Status:    "open",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return e.Repo.InsertOrder(rec)
}
