package model

type Bot struct {
	Id                int64  `json:"id"`
	BotUuid           string `json:"botUuid"`
	Exchange          string `json:"exchange"`
	IsMasterBot       bool   `json:"isMasterBot"`
	IsSwapEnabled     bool   `json:"isSwapEnabled"`
	TradeStackSorting string `json:"tradeStackSorting"`
}
