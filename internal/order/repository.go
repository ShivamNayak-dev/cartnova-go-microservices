package order

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("order not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, userID int64, items []OrderItem, total float64) (*Order, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	order := &Order{
		UserID:      userID,
		TotalAmount: total,
		Status:      "CREATED",
		Items:       items,
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO orders(user_id, total_amount, status)
		VALUES($1, $2, $3)
		RETURNING id, created_at, updated_at
	`, userID, total, order.Status).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return nil, err
	}

	for index := range order.Items {
		item := &order.Items[index]
		err := tx.QueryRow(ctx, `
			INSERT INTO order_items(order_id, product_id, quantity, price)
			VALUES($1, $2, $3, $4)
			RETURNING id
		`, order.ID, item.ProductID, item.Quantity, item.Price).Scan(&item.ID)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return order, nil
}

func (r *Repository) FindByID(ctx context.Context, id int64) (*Order, error) {
	order := &Order{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, total_amount, status, created_at, updated_at
		FROM orders
		WHERE id = $1
	`, id).Scan(&order.ID, &order.UserID, &order.TotalAmount, &order.Status, &order.CreatedAt, &order.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, product_id, quantity, price
		FROM order_items
		WHERE order_id = $1
		ORDER BY id
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	order.Items = make([]OrderItem, 0)
	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(&item.ID, &item.ProductID, &item.Quantity, &item.Price); err != nil {
			return nil, err
		}
		order.Items = append(order.Items, item)
	}

	return order, rows.Err()
}

func (r *Repository) FindByUserID(ctx context.Context, userID int64) ([]Order, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, total_amount, status, created_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]Order, 0)
	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.ID, &order.UserID, &order.TotalAmount, &order.Status, &order.CreatedAt, &order.UpdatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

func (r *Repository) UpdateStatus(ctx context.Context, id int64, status string) error {
	result, err := r.db.Exec(ctx, `
		UPDATE orders
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`, status, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) MarkEventProcessed(ctx context.Context, eventID string) (bool, error) {
	result, err := r.db.Exec(ctx, `
		INSERT INTO processed_events(event_id)
		VALUES($1)
		ON CONFLICT DO NOTHING
	`, eventID)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() == 1, nil
}
