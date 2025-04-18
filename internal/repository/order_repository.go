package repository

import (
	"bybit-bot/internal/model"
	"database/sql"
	"fmt"
	"time"
)

// OrderRepository описывает интерфейс для работы с заказами.
type OrderRepository interface {
	InsertOrder(order *model.Order) error
	UpdateOrder(order *model.Order) error
	FindOrderByID(orderID string) (*model.Order, error)
	FindOrdersBySymbol(symbol string) ([]*model.Order, error)
}

// orderRepository — реализация OrderRepository.
type orderRepository struct {
	db *sql.DB
}

// NewOrderRepository возвращает новую реализацию OrderRepository.
func NewOrderRepository(db *sql.DB) OrderRepository {
	return &orderRepository{
		db: db,
	}
}

// InsertOrder вставляет новый заказ в базу данных.
func (r *orderRepository) InsertOrder(order *model.Order) error {
	query := `
		INSERT INTO orders 
		(order_id, symbol, side, order_type, price, quantity, stop_loss, take_profit, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	now := time.Now()
	order.CreatedAt = now
	order.UpdatedAt = now

	_, err := r.db.Exec(query, order.OrderID, order.Symbol, order.Side, order.OrderType,
		order.Price, order.Quantity, order.StopLoss, order.TakeProfit, order.Status, order.CreatedAt, order.UpdatedAt)
	if err != nil {
		return fmt.Errorf("InsertOrder: %w", err)
	}
	return nil
}

// UpdateOrder обновляет данные заказа в базе данных.
func (r *orderRepository) UpdateOrder(order *model.Order) error {
	query := `
		UPDATE orders 
		SET symbol = $1, side = $2, order_type = $3, price = $4, quantity = $5, stop_loss = $6, take_profit = $7, status = $8, updated_at = $9
		WHERE order_id = $10
	`
	order.UpdatedAt = time.Now()
	_, err := r.db.Exec(query, order.Symbol, order.Side, order.OrderType, order.Price,
		order.Quantity, order.StopLoss, order.TakeProfit, order.Status, order.UpdatedAt, order.OrderID)
	if err != nil {
		return fmt.Errorf("UpdateOrder: %w", err)
	}
	return nil
}

// FindOrderByID возвращает заказ по идентификатору.
func (r *orderRepository) FindOrderByID(orderID string) (*model.Order, error) {
	query := `
		SELECT order_id, symbol, side, order_type, price, quantity, stop_loss, take_profit, status, created_at, updated_at
		FROM orders
		WHERE order_id = $1
	`
	row := r.db.QueryRow(query, orderID)

	var order model.Order
	err := row.Scan(&order.OrderID, &order.Symbol, &order.Side, &order.OrderType,
		&order.Price, &order.Quantity, &order.StopLoss, &order.TakeProfit, &order.Status, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("FindOrderByID: %w", err)
	}
	return &order, nil
}

// FindOrdersBySymbol возвращает список заказов для указанной торговой пары.
func (r *orderRepository) FindOrdersBySymbol(symbol string) ([]*model.Order, error) {
	query := `
		SELECT order_id, symbol, side, order_type, price, quantity, stop_loss, take_profit, status, created_at, updated_at
		FROM orders
		WHERE symbol = $1
	`
	rows, err := r.db.Query(query, symbol)
	if err != nil {
		return nil, fmt.Errorf("FindOrdersBySymbol: %w", err)
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var order model.Order
		err := rows.Scan(&order.OrderID, &order.Symbol, &order.Side, &order.OrderType,
			&order.Price, &order.Quantity, &order.StopLoss, &order.TakeProfit, &order.Status, &order.CreatedAt, &order.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("FindOrdersBySymbol: %w", err)
		}
		orders = append(orders, &order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("FindOrdersBySymbol: %w", err)
	}
	return orders, nil
}
