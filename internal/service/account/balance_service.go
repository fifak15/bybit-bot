package account

import (
	"bybit-bot/internal/client"
	"bybit-bot/internal/model"
	"bybit-bot/internal/repository"
	"bybit-bot/internal/utils"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bybit"
	"log"
	"strconv"
	"time"
)

type BalanceService struct {
	Bybit            *client.ByBit
	WalletRepository *repository.WalletRepository
	Format           *utils.Formatter
}

func ParseAvailableFunds(balance *bybit.WalletBalance, coin string) float64 {
	for _, wallet := range balance.List {
		for _, c := range wallet.Coin {
			if c.Coin.String() == coin {
				available, err := strconv.ParseFloat(c.WalletBalance.String(), 64)
				if err != nil {
					log.Printf("Ошибка парсинга баланса для %s: %v", coin, err)
					return 0.0
				}
				return available
			}
		}
	}
	return 0.0
}

func CheckBalanceAndSave(balance *bybit.WalletBalance, coin string, requiredFunds float64, walletRepo *repository.WalletRepository) bool {
	availableFunds := ParseAvailableFunds(balance, coin)
	totalMarginBalance := 0.0
	totalWalletBalance := 0.0
	if len(balance.List) > 0 {
		totalMarginBalance = balance.List[0].TotalMarginBalance.Float64()
		totalWalletBalance = balance.List[0].TotalWalletBalance.Float64()
	}

	walletInfo := &model.WalletInfoRep{
		Coin:               coin,
		WalletBalance:      availableFunds,
		TotalMarginBalance: totalMarginBalance,
		TotalWalletBalance: totalWalletBalance,
		RecordedAt:         time.Now(),
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := walletRepo.SaveWalletInfo(walletInfo); err != nil {
		log.Printf("Ошибка сохранения баланса для %s: %v", coin, err)
		return false
	}

	if availableFunds < requiredFunds {
		log.Printf("Недостаточно средств: доступно %.2f %s, требуется минимум %.2f %s",
			availableFunds, coin, requiredFunds, coin)
		return false
	}
	return true
}

func (b *BalanceService) CheckBalance(buyPrice, buyQuantity, sellPrice, sellQuantity float64) bool {
	buyOrderCost := buyPrice * buyQuantity
	sellOrderCost := sellPrice * sellQuantity

	balanceData, err := b.Bybit.GetBalance()
	if err != nil {
		log.Printf("Ошибка получения баланса: %v", err)
		return false
	}
	if !CheckBalanceAndSave(balanceData, "USDT", buyOrderCost, b.WalletRepository) {
		return false
	}
	if !CheckBalanceAndSave(balanceData, "USDT", sellOrderCost, b.WalletRepository) {
		return false
	}
	return true
}

func (b *BalanceService) CheckAndFormatPrices(category, symbol string, buyPrice, sellPrice, stopLossBuy, stopLossSell,
	takeProfitBuy, takeProfitSell float64) (model.TradeLimits, float64, float64, float64, float64, float64, float64, bool) {

	tradeLimit, err := b.Bybit.GetTradeLimitsViaInstruments(category, symbol)
	if err != nil {
		log.Printf("Ошибка получения торговых лимитов: %v", err)
		return tradeLimit, 0, 0, 0, 0, 0, 0, false
	}

	formatPrice := func(price float64) float64 {
		return b.Format.FormatPrice(tradeLimit, price)
	}

	buyPriceF := formatPrice(buyPrice)
	sellPriceF := formatPrice(sellPrice)
	stopLossBuyF := formatPrice(stopLossBuy)
	stopLossSellF := formatPrice(stopLossSell)
	takeProfitBuyF := formatPrice(takeProfitBuy)
	takeProfitSellF := formatPrice(takeProfitSell)

	return tradeLimit, buyPriceF, sellPriceF, stopLossBuyF, stopLossSellF, takeProfitBuyF, takeProfitSellF, true
}
