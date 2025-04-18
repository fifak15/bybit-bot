package model

import "time"

type WalletBalanceResponse struct {
	RetCode int     `json:"retCode"`
	RetMsg  string  `json:"retMsg"`
	Result  *Result `json:"result"`
	Time    int64   `json:"time"`
}

type Result struct {
	List []WalletInfo `json:"list"`
}

type WalletInfo struct {
	Coin []CoinBalance `json:"coin"`
}

type CoinBalance struct {
	Coin          string  `json:"coin"`
	WalletBalance *string `json:"walletBalance"`
}

type WalletInfoRep struct {
	ID                 int       `json:"id"`                   // первичный ключ
	Coin               string    `json:"coin"`                 // например, "USDT"
	WalletBalance      float64   `json:"wallet_balance"`       // текущий баланс
	TotalMarginBalance float64   `json:"total_margin_balance"` // маржинальный баланс (опционально)
	TotalWalletBalance float64   `json:"total_wallet_balance"` // общий баланс (опционально)
	RecordedAt         time.Time `json:"recorded_at"`          // время фиксации баланса
	CreatedAt          time.Time `json:"created_at"`           // время создания записи
	UpdatedAt          time.Time `json:"updated_at"`           // время последнего обновления записи
}
