package model

import "github.com/thrasher-corp/gocryptotrader/types"

type Trade struct {
	AggregateTradeId int64          `json:"a,int"`
	Price            float64        `json:"p,string"`
	Symbol           string         `json:"s"`
	Quantity         float64        `json:"q,string"`
	IsBuyerMaker     bool           `json:"m,bool"` // IsBuyerMaker = true -> SELL / IsBuyerMaker = false -> BUY
	Timestamp        TimestampMilli `json:"T,int"`
	Ignore           bool           `json:"M,bool"`
}

func (c *Trade) GetOperation() string {
	if c.IsSell() {
		return "SELL"
	}

	return "BUY"
}

type TradingRequirements struct {
	MakerFee types.Number `json:"maker_fee"`
	TakerFee types.Number `json:"taker_fee"`
}

func (c *Trade) IsSell() bool {
	return c.IsBuyerMaker == true
}

func (c *Trade) IsBuy() bool {
	return c.IsBuyerMaker == false
}

type TradeVolume struct {
	Symbol     string         `json:"symbol"`
	Timestamp  TimestampMilli `json:"timestamp"`
	PeriodFrom TimestampMilli `json:"periodFrom"`
	PeriodTo   TimestampMilli `json:"periodTo"`
	BuyQty     float64        `json:"buyQty"`
	SellQty    float64        `json:"sellQty"`
}
