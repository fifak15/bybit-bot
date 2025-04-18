package model

type KlineMessage struct {
	Topic string      `json:"topic"` // например, "kline.1.BTCUSDT"
	Type  string      `json:"type"`  // "snapshot" или "delta"
	Ts    int64       `json:"ts"`
	Data  []KlineData `json:"data"`
}
