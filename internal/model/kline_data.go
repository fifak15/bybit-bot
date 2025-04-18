package model

import "time"

type KlineData struct {
	Start     int64   `json:"start"`           // время начала свечи (миллисекунды)
	End       int64   `json:"end"`             // время окончания свечи
	Interval  string  `json:"interval"`        // интервал (например, "1")
	Open      float64 `json:"open,string"`     // цена открытия (передается как строка)
	Close     float64 `json:"close,string"`    // цена закрытия
	High      float64 `json:"high,string"`     // максимальная цена
	Low       float64 `json:"low,string"`      // минимальная цена
	Volume    float64 `json:"volume,string"`   // объём
	Turnover  float64 `json:"turnover,string"` // оборот
	Confirm   bool    `json:"confirm"`         // подтверждено или нет
	Timestamp int64   `json:"timestamp"`       // временная метка свечи
	Symbol    string  `json:"s,omitempty"`
	UpdatedAt int64   `json:"updatedAt"`
}

const PriceValidSecondes = 30

func (k *KlineData) IsPriceExpiredesas() bool {
	return (time.Now().Unix() - (k.UpdatedAt)) > PriceValidSecondes
}

type KlineDto struct {
	StartTime  string `json:"startTime"`
	OpenPrice  string `json:"openPrice"`
	HighPrice  string `json:"highPrice"`
	LowPrice   string `json:"lowPrice"`
	ClosePrice string `json:"closePrice"`
	Volume     string `json:"volume"`
}
