package repository

import (
	"bybit-bot/internal/model"
	"database/sql"
	"fmt"
	"time"
)

type WalletRepository struct {
	db *sql.DB
}

func NewWalletRepository(db *sql.DB) *WalletRepository {
	return &WalletRepository{db: db}
}

func (r *WalletRepository) InsertWalletInfo(info *model.WalletInfoRep) error {
	query := `
		INSERT INTO wallet_info (coin, wallet_balance, total_margin_balance, total_wallet_balance, recorded_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id
	`
	err := r.db.QueryRow(query,
		info.Coin,
		info.WalletBalance,
		info.TotalMarginBalance,
		info.TotalWalletBalance,
		info.RecordedAt,
	).Scan(&info.ID)
	if err != nil {
		return fmt.Errorf("failed to insert wallet info: %w", err)
	}
	return nil
}

// UpdateWalletInfo обновляет существующую запись баланса по идентификатору.
func (r *WalletRepository) UpdateWalletInfo(info *model.WalletInfoRep) error {
	query := `
		UPDATE wallet_info
		SET coin = $1, wallet_balance = $2, total_margin_balance = $3,
		    total_wallet_balance = $4, recorded_at = $5, updated_at = NOW()
		WHERE id = $6
	`
	_, err := r.db.Exec(query,
		info.Coin,
		info.WalletBalance,
		info.TotalMarginBalance,
		info.TotalWalletBalance,
		info.RecordedAt,
		info.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update wallet info: %w", err)
	}
	return nil
}

// GetLatestWalletInfo возвращает последнюю запись баланса для заданной валюты.
func (r *WalletRepository) GetLatestWalletInfo(coin string) (*model.WalletInfoRep, error) {
	query := `
		SELECT id, coin, wallet_balance, total_margin_balance, total_wallet_balance, recorded_at, created_at, updated_at
		FROM wallet_info
		WHERE coin = $1
		ORDER BY recorded_at DESC
		LIMIT 1
	`
	row := r.db.QueryRow(query, coin)
	var info model.WalletInfoRep
	err := row.Scan(
		&info.ID,
		&info.Coin,
		&info.WalletBalance,
		&info.TotalMarginBalance,
		&info.TotalWalletBalance,
		&info.RecordedAt,
		&info.CreatedAt,
		&info.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil //
		}
		return nil, fmt.Errorf("failed to get wallet info: %w", err)
	}
	return &info, nil
}

func (r *WalletRepository) GetLatestWalletBalance(walletBalance float64) (*model.WalletInfoRep, error) {
	query := `
		SELECT id, coin, wallet_balance, total_margin_balance, total_wallet_balance, recorded_at, created_at, updated_at
		FROM wallet_info
		WHERE wallet_balance = $1
		ORDER BY recorded_at DESC
		LIMIT 1
	`
	row := r.db.QueryRow(query, walletBalance)
	var info model.WalletInfoRep
	err := row.Scan(
		&info.ID,
		&info.Coin,
		&info.WalletBalance,
		&info.TotalMarginBalance,
		&info.TotalWalletBalance,
		&info.RecordedAt,
		&info.CreatedAt,
		&info.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil //
		}
		return nil, fmt.Errorf("failed to get wallet info: %w", err)
	}
	return &info, nil
}

func (r *WalletRepository) SaveWalletInfo(info *model.WalletInfoRep) error {
	existing, err := r.GetLatestWalletBalance(info.WalletBalance)
	if err != nil {
		return fmt.Errorf("failed to get latest wallet Balance: %w", err)
	}
	info.RecordedAt = time.Now()
	if existing == nil {
		return r.InsertWalletInfo(info)
	} else {
		info.ID = existing.ID
		return r.UpdateWalletInfo(info)
	}
}
