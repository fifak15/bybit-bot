package model

type MarketDataDto struct {
	MarketDataResult MarketDataResult `json:"result"`
}

type MarketDataResult struct {
	List []MarketDataItem `json:"list"`
}

type MarketDataItem struct {
	Symbol    string `json:"symbol"`
	LastPrice string `json:"lastPrice"`
	Volume24h string `json:"volume24h"`
}
