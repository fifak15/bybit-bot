package main

import (
	"bybit-bot/internal/client"
	"bybit-bot/internal/repository"
	"bybit-bot/internal/service/account"
	"bybit-bot/internal/service/event"
	"bybit-bot/internal/service/exchange"
	"bybit-bot/internal/service/strategy"
	"bybit-bot/internal/utils"
	"database/sql"
	_ "github.com/lib/pq"
	"io"
	"log"
	"os"
	"time"
)

func main() {
	// 1. Подключаемся к PostgreSQL
	dsn := "postgres://postgres:1234@localhost:5433/postgres?sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer db.Close()

	// Проверим соединение
	if err := db.Ping(); err != nil {
		log.Fatalf("БД недоступна: %v", err)
	}
	log.Println("Подключение к PostgreSQL установлено.")

	// 2. Создаём репозиторий ордеров
	orderRepo := repository.NewOrderRepository(db)

	walletRepo := repository.NewWalletRepository(db)

	// 3. Инициализируем WSListener
	wsURL := "wss://stream.bybit.com/v5/public/linear"
	wsListener, err := event.NewWSListener(wsURL, nil)
	if err != nil {
		log.Fatalf("Ошибка подключения к WS: %v", err)
	}
	go wsListener.ListenAll()

	channels := []string{"orderbook.50.BTCUSDT", "kline.1.BTCUSDT"}
	if err := wsListener.SubscribeChannels(channels); err != nil {
		log.Fatalf("Ошибка подписки на каналы: %v", err)
	}

	// Даем время на получение первых данных
	time.Sleep(10 * time.Second)

	// 2. Инициализируем ByBit клиент с вашими API-ключом и секретом (используйте тестовые ключи)
	bybitClient := client.NewByBit("Cv6vQhpZDnSFROonKx", "aIJarBdglaBBDx7VHFFW9x0lKWEF4ez7mupL")
	signalChan := strategy.NewSignalDetector()
	balanceService := &account.BalanceService{
		Bybit:            bybitClient,
		WalletRepository: walletRepo,
	}

	priceCalculator := &exchange.PriceCalculator{
		OrderRepository:  orderRepo,
		WSListener:       wsListener,
		WalletRepository: walletRepo,
	}

	// 7. Инициализируем Formatter (для округления цен)
	formatter := &utils.Formatter{}

	// 8. Собираем стратегию маркет-мейкинга
	strategyVPA := &strategy.VPAScalping{
		OrderRepository:  orderRepo,
		WalletRepository: walletRepo,
		BalanceService:   balanceService,
		Formatter:        formatter,
		Bybit:            bybitClient,
		SignalDetector:   signalChan,
		PriceCalculator:  priceCalculator,
		WSListener:       wsListener,
		StopLossPercent:  0.005, // например, 0.5% за порог
	}

	log.Println("Стратегия vpa_scalping запущена, ожидаем данных...")

	for {
		strategyVPA.Make("BTCUSDT", "linear")
		time.Sleep(30 * time.Second)
	}

}
func init() {
	// Открываем (или создаём) файл в текущей папке
	f, err := os.OpenFile("vpa_scalping.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		log.Fatalf("не удалось открыть лог-файл: %v", err)
	}

	// Чтобы логи шли одновременно и в консоль, и в файл:
	mw := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mw)

	// (опционально) добавить префикс и флаги времени:
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
}
