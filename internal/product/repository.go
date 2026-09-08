package product

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("product not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, product *Product) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO products(name, description, price, category_id)
		VALUES($1, $2, $3, $4)
		RETURNING id, status, created_at, updated_at
	`, product.Name, product.Description, product.Price, product.CategoryID).
		Scan(&product.ID, &product.Status, &product.CreatedAt, &product.UpdatedAt)
}

func (r *Repository) FindByID(ctx context.Context, id int64) (*Product, error) {
	product := &Product{}

	err := r.db.QueryRow(ctx, `
		SELECT id, name, description, price, category_id, status, created_at, updated_at
		FROM products
		WHERE id = $1
	`, id).Scan(
		&product.ID,
		&product.Name,
		&product.Description,
		&product.Price,
		&product.CategoryID,
		&product.Status,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return product, err
}

func (r *Repository) FindAll(ctx context.Context, categoryID *int64, search string) ([]Product, error) {
	query := `
		SELECT id, name, description, price, category_id, status, created_at, updated_at
		FROM products
	`
	args := []any{}
	conditions := make([]string, 0, 2)
	if categoryID != nil {
		args = append(args, *categoryID)
		conditions = append(conditions, fmt.Sprintf("category_id = $%d", len(args)))
	}
	if strings.TrimSpace(search) != "" {
		args = append(args, "%"+strings.TrimSpace(search)+"%")
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", len(args), len(args)))
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY id DESC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]Product, 0)
	for rows.Next() {
		var product Product
		if err := rows.Scan(
			&product.ID,
			&product.Name,
			&product.Description,
			&product.Price,
			&product.CategoryID,
			&product.Status,
			&product.CreatedAt,
			&product.UpdatedAt,
		); err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	return products, rows.Err()
}

func (r *Repository) Update(ctx context.Context, id int64, product *Product) error {
	result, err := r.db.Exec(ctx, `
		UPDATE products
		SET name = $1,
			description = $2,
			price = $3,
			category_id = $4,
			status = $5,
			updated_at = NOW()
		WHERE id = $6
	`, product.Name, product.Description, product.Price, product.CategoryID, product.Status, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	product.ID = id
	return nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.Exec(ctx, "DELETE FROM products WHERE id = $1", id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
