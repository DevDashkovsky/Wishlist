package repository

import (
	"context"
	"errors"
	"wishlist-api/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) CreateUser(ctx context.Context, u *domain.User) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash)
		 VALUES ($1, $2)
		 RETURNING id, created_at`,
		u.Email, u.PasswordHash,
	).Scan(&u.ID, &u.CreatedAt)
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, created_at
		 FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
