package utils

import "bybit-bot/internal/model"

func СalculateWeightedMidPrice(orderBook *model.OrderbookData, levels int) (float64, error) {
	if levels <= 0 || len(orderBook.Bids) == 0 || len(orderBook.Asks) == 0 {
		return 0.0, nil
	}

	totalBidValue, totalBidQty := 0.0, 0.0
	nBids := levels
	if len(orderBook.Bids) < levels {
		nBids = len(orderBook.Bids)
	}
	for i := 0; i < nBids; i++ {
		bid := orderBook.Bids[i]
		totalBidValue += bid.Price * bid.Size
		totalBidQty += bid.Size
	}
	weightedBid := 0.0
	if totalBidQty > 0 {
		weightedBid = totalBidValue / totalBidQty
	}

	totalAskValue, totalAskQty := 0.0, 0.0
	nAsks := levels
	if len(orderBook.Asks) < levels {
		nAsks = len(orderBook.Asks)
	}
	for i := 0; i < nAsks; i++ {
		ask := orderBook.Asks[i]
		totalAskValue += ask.Price * ask.Size
		totalAskQty += ask.Size
	}
	weightedAsk := 0.0
	if totalAskQty > 0 {
		weightedAsk = totalAskValue / totalAskQty
	}

	return (weightedBid + weightedAsk) / 2, nil
}
