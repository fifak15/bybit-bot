package model

type KlineResponseDto struct {
	Category string      `json:"category"`
	Symbol   string      `json:"symbol"`
	Result   KlineResult `json:"result"`
}

type KlineResult struct {
	List [][]string `json:"list"`
}
