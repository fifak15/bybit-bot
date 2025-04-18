package model

type TradeOrderRequest struct {
	Category string `json:"category"`
	Symbol   string `json:"symbol"`
	Limit    int    `json:"limit"`
}
