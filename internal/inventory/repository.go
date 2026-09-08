package inventory

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("inventory not found")
var ErrInsufficientStock = errors.New("insufficient stock")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateOrUpdate(ctx context.Context, productID int64, quantity int) (*Inventory, error) {
	inventory := &Inventory{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO inventory(product_id, available_quantity)
		VALUES($1, $2)
		ON CONFLICT(product_id)
		DO UPDATE SET available_quantity = inventory.available_quantity + EXCLUDED.available_quantity,
		              updated_at = NOW()
		RETURNING product_id, available_quantity, reserved_quantity
	`, productID, quantity).Scan(&inventory.ProductID, &inventory.AvailableQuantity, &inventory.ReservedQuantity)
	return inventory, err
}

func (r *Repository) ReserveOrder(ctx context.Context, orderID int64, items []ReservationItem) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, item := range items {
		var alreadyReserved bool
		err := tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM reservations
				WHERE order_id = $1
				  AND product_id = $2
				  AND status = 'RESERVED'
			)
		`, orderID, item.ProductID).Scan(&alreadyReserved)
		if err != nil {
			return err
		}
		if alreadyReserved {
			continue
		}

		var available int
		err = tx.QueryRow(ctx, `
			SELECT available_quantity
			FROM inventory
			WHERE product_id = $1
			FOR UPDATE
		`, item.ProductID).Scan(&available)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if available < item.Quantity {
			return ErrInsufficientStock
		}

		_, err = tx.Exec(ctx, `
			UPDATE inventory
			SET available_quantity = available_quantity - $1,
				reserved_quantity = reserved_quantity + $1,
				updated_at = NOW()
			WHERE product_id = $2
		`, item.Quantity, item.ProductID)
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO reservations(order_id, product_id, quantity, status)
			VALUES($1, $2, $3, 'RESERVED')
		`, orderID, item.ProductID, item.Quantity)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) Get(ctx context.Context, productID int64) (*Inventory, error) {
	inventory := &Inventory{}
	err := r.db.QueryRow(ctx, `
		SELECT product_id, available_quantity, reserved_quantity
		FROM inventory
		WHERE product_id = $1
	`, productID).Scan(&inventory.ProductID, &inventory.AvailableQuantity, &inventory.ReservedQuantity)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return inventory, err
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
