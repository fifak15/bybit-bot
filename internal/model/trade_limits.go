package model

// TradeLimits описывает торговые лимиты (параметры  торговой пары)
type TradeLimits struct {
	Symbol      string  `json:"symbol"`
	MinQuantity float64 `json:"min_quantity"` // Минимальное количество ордера
	MaxQuantity float64 `json:"max_quantity"` // Максимальное количество ордера
	LotSize     float64 `json:"lot_size"`     // Размер лота
	StepSize    float64 `json:"step_size"`    // Шаг изменения количества
	TickSize    float64 `json:"tick_size"`    // Шаг изменения цены
	MinNotional float64 `json:"min_notional"` // Минимальная стоимость ордера
	MaxNotional float64 `json:"max_notional"` // Максимальная стоимость ордера
}
