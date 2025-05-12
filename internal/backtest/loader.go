package backtest

import (
	"bybit-bot/internal/model"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	_ "strconv"
	_ "time"
)

type Candle struct {
	Timestamp int64
	Open      string
	High      string
	Low       string
	Close     string
	Volume    string
}

type Response struct {
	RetCode int `json:"retCode"`
	Result  struct {
		List [][]interface{} `json:"list"`
	} `json:"result"`
}

func LoadKlinesFromCSV(filePath string) ([]model.KlineData, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var klines []model.KlineData
	for i, row := range rows {
		if i == 0 {
			continue
		}
		ts, _ := strconv.ParseInt(row[0], 10, 64)
		open, _ := strconv.ParseFloat(row[1], 64)
		high, _ := strconv.ParseFloat(row[2], 64)
		low, _ := strconv.ParseFloat(row[3], 64)
		closePrice, _ := strconv.ParseFloat(row[4], 64)
		volume, _ := strconv.ParseFloat(row[5], 64)

		klines = append(klines, model.KlineData{
			Timestamp: ts,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closePrice,
			Volume:    volume,
		})
	}

	return klines, nil
}

func DownLoadKlinesFromBybit() error {
	symbol := "BTCUSDT"
	interval := "30" // 30m
	limit := 1000

	endpoint := fmt.Sprintf("https://api.bybit.com/v5/market/kline?category=linear&symbol=%s&interval=%s&limit=%d", symbol, interval, limit)
	resp, err := http.Get(endpoint)
	if err != nil {
		return fmt.Errorf("ошибка HTTP-запроса: %v", err)
	}
	defer resp.Body.Close()

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("ошибка декодирования JSON: %v", err)
	}

	file, err := os.Create("bybit_klines.csv")
	if err != nil {
		return fmt.Errorf("ошибка создания файла: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"timestamp", "open", "high", "low", "close", "volume"})

	for _, row := range result.Result.List {
		timestamp := row[0].(string)
		open := row[1].(string)
		high := row[2].(string)
		low := row[3].(string)
		close := row[4].(string)
		volume := row[5].(string)

		writer.Write([]string{timestamp, open, high, low, close, volume})
	}

	fmt.Println("Сохранено в bybit_klines.csv")
	fmt.Println("CSV сохранён:", "bybit_klines.csv")
	return nil
}
