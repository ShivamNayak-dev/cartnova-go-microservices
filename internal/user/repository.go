package user

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("user not found")
var ErrEmailExists = errors.New("email already exists")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, user *User) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO users(name, email, password_hash, role)
		VALUES($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`, user.Name, user.Email, user.PasswordHash, user.Role).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return ErrEmailExists
		}
		return err
	}

	return nil
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}

	err := r.db.QueryRow(ctx, `
		SELECT id, name, email, password_hash, role, created_at, updated_at
		FROM users
		WHERE email = $1
	`, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return user, err
}

func (r *Repository) FindByID(ctx context.Context, id int64) (*User, error) {
	user := &User{}

	err := r.db.QueryRow(ctx, `
		SELECT id, name, email, password_hash, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return user, err
}

func isUniqueViolation(err error) bool {
	var pgError *pgconn.PgError
	if !errors.As(err, &pgError) {
		return false
	}
	return pgError.Code == "23505"
}

func (r *Repository) UpdateName(ctx context.Context, id int64, name string) error {
	result, err := r.db.Exec(ctx, `
		UPDATE users
		SET name = $1, updated_at = NOW()
		WHERE id = $2
	`, name, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
