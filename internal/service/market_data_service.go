package service

import (
	"bybit-bot/internal/model"
	_ "context"
	"encoding/json"
	"fmt"
	_ "github.com/mitchellh/mapstructure"
	"github.com/wuhewuhe/bybit.go.api"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

type MarketDataService struct {
	client *bybit_connector.Client
}

func NewMarketDataService() *MarketDataService {
	client := bybit_connector.NewBybitHttpClient("Cv6vQhpZDnSFROonKx", "aIJarBdglaBBDx7VHFFW9x0lKWEF4ez7mupL", bybit_connector.WithBaseURL("/api/v1/wallet-balance"))
	return &MarketDataService{client: client}
}

func (s *MarketDataService) GetServerTimeMillis() int64 {
	return time.Now().UnixMilli()
}

func (s *MarketDataService) GetHistoricalData(symbol, interval string, limit int, category string) ([]model.KlineDto, error) {

	url := fmt.Sprintf("https://api.bybit.com/v5/market/kline?category=%s&symbol=%s&interval=%s&limit=%d", category, symbol, interval, limit)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	log.Printf("Response Status Code: %d", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	log.Printf("Raw response body: %s", body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch data: %s", resp.Status)
	}

	var response model.KlineResponseDto
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	log.Printf("Decoded response: %+v", response)

	if response.Result.List == nil || len(response.Result.List) == 0 {
		return nil, fmt.Errorf("no historical data available for the specified symbol and interval")
	}

	var klines []model.KlineDto
	for _, klineArray := range response.Result.List {
		startTimeMillis, _ := strconv.ParseInt(klineArray[0], 10, 64)
		startTime := time.Unix(startTimeMillis/1000, 0).Format("2006-01-02 15:04:05")

		klines = append(klines, model.KlineDto{
			StartTime:  startTime,
			OpenPrice:  klineArray[1],
			HighPrice:  klineArray[2],
			LowPrice:   klineArray[3],
			ClosePrice: klineArray[4],
			Volume:     klineArray[5],
		})
	}

	return klines, nil
}
