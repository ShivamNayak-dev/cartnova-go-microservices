package product

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Category struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type CategoryRequest struct {
	Name string `json:"name"`
}

var ErrCategoryNotFound = errors.New("category not found")

type CategoryRepository struct {
	db *pgxpool.Pool
}

func NewCategoryRepository(db *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) Create(ctx context.Context, name string) (*Category, error) {
	category := &Category{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO categories(name)
		VALUES($1)
		RETURNING id, name, created_at
	`, name).Scan(&category.ID, &category.Name, &category.CreatedAt)
	return category, err
}

func (r *CategoryRepository) FindAll(ctx context.Context) ([]Category, error) {
	rows, err := r.db.Query(ctx, "SELECT id, name, created_at FROM categories ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := make([]Category, 0)
	for rows.Next() {
		var category Category
		if err := rows.Scan(&category.ID, &category.Name, &category.CreatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, rows.Err()
}

func (r *CategoryRepository) FindByID(ctx context.Context, id int64) (*Category, error) {
	category := &Category{}
	err := r.db.QueryRow(ctx, "SELECT id, name, created_at FROM categories WHERE id = $1", id).
		Scan(&category.ID, &category.Name, &category.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCategoryNotFound
	}
	return category, err
}

func (r *CategoryRepository) Update(ctx context.Context, id int64, name string) (*Category, error) {
	category := &Category{}
	err := r.db.QueryRow(ctx, `
		UPDATE categories
		SET name = $1
		WHERE id = $2
		RETURNING id, name, created_at
	`, name, id).Scan(&category.ID, &category.Name, &category.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCategoryNotFound
	}
	return category, err
}

func (r *CategoryRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.Exec(ctx, "DELETE FROM categories WHERE id = $1", id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrCategoryNotFound
	}
	return nil
}
