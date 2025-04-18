package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const MinProfitPercent = 0.50

type Percent float64

func (p Percent) IsPositive() bool {
	return float64(p) > 0
}

func (p Percent) Value() float64 {
	return float64(p)
}

func (p Percent) Half() Percent {
	return Percent(float64(p) / 2)
}

func (p Percent) Gt(percent Percent) bool {
	return p.Value() > percent.Value()
}

func (p Percent) Gte(percent Percent) bool {
	return p.Value() >= percent.Value()
}

func (p Percent) Lte(percent Percent) bool {
	return p.Value() <= percent.Value()
}

func (p Percent) Lt(percent Percent) bool {
	return p.Value() < percent.Value()
}

type ErrorNotification struct {
	BotUuid      string `json:"bot"`
	Stop         bool   `json:"stop"`
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

type TgOrderNotification struct {
	BotUuid   string  `json:"bot"`
	Price     float64 `json:"price"`
	Quantity  float64 `json:"amount"`
	Symbol    string  `json:"symbol"`
	Operation string  `json:"operation"`
	DateTime  string  `json:"dateTime"`
	Details   string  `json:"details"`
}

type ProfitPositionInterface interface {
	GetPositionTime() PositionTime
	GetProfitOptions() ProfitOptions
	GetSymbol() string
	GetExecutedQuantity() float64
	GetPositionQuantityWithSwap() float64
}

type Order struct {
	OrderID    string    `json:"order_id"`
	Symbol     string    `json:"symbol"`
	Side       string    `json:"side"`
	OrderType  string    `json:"order_type"`
	Price      float64   `json:"price"`
	Quantity   float64   `json:"quantity"`
	StopLoss   float64   `json:"stop_loss"`
	TakeProfit float64   `json:"take_profit"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type TradeOrders struct {
	Orders []Order
}

func (o *Order) GetBaseAsset() string {
	return strings.ReplaceAll(o.Symbol, "USDT", "")
}

func (o *Order) GetManualMinClosePrice() float64 {
	return o.Price * (100 + MinProfitPercent) / 100
}

func (o *Order) IsClosed() bool {
	return o.Status == "closed"
}

func (o *Order) IsOpened() bool {
	return o.Status == "opened"
}

func (o Order) GetSymbol() string {
	return o.Symbol
}

type BuyPrice struct {
	Error error
	Price float64
}

type Signal struct {
	Symbol          string  `json:"symbol"`
	BuyPrice        float64 `json:"buyPrice"`
	Percent         Percent `json:"percent"`
	ExpireTimestamp int64   `json:"expireTimestamp"`
	Exchange        string  `json:"exchange"`
}

type Position struct {
	Symbol              string                `json:"symbol"`
	KLine               KLine                 `json:"kLine"`
	Order               Order                 `json:"order"`
	Percent             Percent               `json:"percent"`
	SellPrice           float64               `json:"sellPrice"`
	PredictedPrice      float64               `json:"predictedPrice"`
	PriceChangeSpeedAvg float64               `json:"priceChangeSpeedAvg"`
	IsPriceExpired      bool                  `json:"isPriceExpired"`
	Profit              float64               `json:"profit"`
	TargetProfit        float64               `json:"targetProfit"`
	Interpolation       Interpolation         `json:"interpolation"`
	OrigQty             float64               `json:"origQty"`
	ExecutedQty         float64               `json:"executedQty"`
	ManualOrderConfig   ManualOrderConfig     `json:"manualOrderConfig"`
	PositionTime        PositionTime          `json:"positionTime"`
	CloseStrategy       PositionCloseStrategy `json:"closeStrategy"`
	ManualOrder         *ManualOrder          `json:"manualOrder"`
	IsEnabled           bool                  `json:"isEnabled"`
	CanSell             bool                  `json:"canSell"`
	CanExtraBuy         bool                  `json:"canExtraBuy"`
	Capitalization      Capitalization        `json:"capitalization"`
}

type PositionCloseStrategy struct {
	MinClosePrice    float64 `json:"minClosePrice"`
	MinProfitPercent Percent `json:"minProfitPercent"`
}

type ManualOrder struct {
	Operation string  `json:"operation"`
	Price     float64 `json:"price"`
	Symbol    string  `json:"symbol"`
	BotUuid   string  `json:"botUuid"`
	Ttl       int64   `json:"ttl"`
}

type ManualOrderConfig struct {
	PriceStep     float64 `json:"priceStep"`
	MinClosePrice float64 `json:"minSellPrice"`
}

type PendingOrder struct {
	Symbol         string        `json:"symbol"`
	KLine          KLine         `json:"kLine"`
	PredictedPrice float64       `json:"predictedPrice"`
	Interpolation  Interpolation `json:"interpolation"`
	IsRisky        bool          `json:"isRisky"`
}

type Interpolation struct {
	Asset                string  `json:"asset"`
	BtcInterpolationUsdt float64 `json:"btcInterpolationUsdt"`
	EthInterpolationUsdt float64 `json:"ethInterpolationUsdt"`
}

type ExtraChargeOptions []ExtraChargeOption

type ExtraChargeOption struct {
	Index      int64   `json:"index"`
	Percent    Percent `json:"percent"`
	AmountUsdt float64 `json:"amountUsdt"`
}

func (e *ExtraChargeOptions) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), &e)
}

func (e ExtraChargeOptions) Value() (driver.Value, error) {
	jsonV, err := json.Marshal(e)
	return string(jsonV), err
}

type ProfitOptions []ProfitOption

const ProfitOptionUnitMinute = "i"
const ProfitOptionUnitHour = "h"
const ProfitOptionUnitDay = "d"
const ProfitOptionUnitMonth = "m"

type ProfitOption struct {
	Index           int64   `json:"index"`
	IsTriggerOption bool    `json:"isTriggerOption"`
	OptionValue     float64 `json:"optionValue"`
	OptionUnit      string  `json:"optionUnit"`
	OptionPercent   Percent `json:"optionPercent"`
}

func (p *ProfitOptions) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), &p)
}
func (p ProfitOptions) Value() (driver.Value, error) {
	jsonV, err := json.Marshal(p)
	return string(jsonV), err
}
func (p ProfitOption) IsMinutely() bool {
	return p.OptionUnit == ProfitOptionUnitMinute
}
func (p ProfitOption) IsHourly() bool {
	return p.OptionUnit == ProfitOptionUnitHour
}
func (p ProfitOption) IsDaily() bool {
	return p.OptionUnit == ProfitOptionUnitDay
}
func (p ProfitOption) IsMonthly() bool {
	return p.OptionUnit == ProfitOptionUnitMonth
}
func (p ProfitOption) GetPositionTime() (PositionTime, error) {
	switch p.OptionUnit {
	case ProfitOptionUnitMinute:
		return PositionTime(p.OptionValue * 60), nil
	case ProfitOptionUnitHour:
		return PositionTime(p.OptionValue * 3600), nil
	case ProfitOptionUnitDay:
		return PositionTime(p.OptionValue * 3600 * 24), nil
	case ProfitOptionUnitMonth:
		return PositionTime(p.OptionValue * 3600 * 24 * 30), nil
	}

	return PositionTime(0.00), errors.New("position time is invalid")
}

type PositionTime int64

func (p PositionTime) GetMinutes() float64 {
	return float64(p) / float64(60)
}
func (p PositionTime) GetHours() float64 {
	return float64(p) / float64(3600)
}
func (p PositionTime) GetDays() float64 {
	return float64(p) / float64(3600*24)
}
func (p PositionTime) GetMonths() float64 {
	return float64(p) / float64(3600*24*30)
}

type Capitalization struct {
	Capitalization float64 `json:"capitalization"`
	MarketPrice    float64 `json:"marketPrice"`
}
