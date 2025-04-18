package model

type TradeHistoryResponse struct {
	RetCode    int                    `json:"retCode"`
	RetMsg     string                 `json:"retMsg"`
	Result     Result                 `json:"result"`
	RetExtInfo map[string]interface{} `json:"retExtInfo"`
	Time       int64                  `json:"time"`
}

type WalletBalanceResult struct {
	NextPageCursor string  `json:"nextPageCursor"`
	Category       string  `json:"category"`
	List           []Trade `json:"list"`
}

type Trades struct {
	Symbol          string  `json:"symbol"`
	OrderType       string  `json:"orderType"`
	UnderlyingPrice string  `json:"underlyingPrice"`
	OrderLinkId     string  `json:"orderLinkId"`
	OrderId         string  `json:"orderId"`
	StopOrderType   string  `json:"stopOrderType"`
	ExecTime        string  `json:"execTime"`
	FeeCurrency     string  `json:"feeCurrency"`
	CreateType      string  `json:"createType"`
	FeeRate         string  `json:"feeRate"`
	TradeIv         string  `json:"tradeIv"`
	BlockTradeId    string  `json:"blockTradeId"`
	MarkPrice       string  `json:"markPrice"`
	ExecPrice       string  `json:"execPrice"`
	OrderQty        string  `json:"orderQty"`
	OrderPrice      string  `json:"orderPrice"`
	ExecValue       string  `json:"execValue"`
	ClosedSize      string  `json:"closedSize"`
	ExecType        string  `json:"execType"`
	Seq             float64 `json:"seq"`
	Side            string  `json:"side"`
	LeavesQty       string  `json:"leavesQty"`
	IsMaker         bool    `json:"isMaker"`
	ExecFee         string  `json:"execFee"`
	ExecId          string  `json:"execId"`
	MarketUnit      string  `json:"marketUnit"`
	ExecQty         string  `json:"execQty"`
}
