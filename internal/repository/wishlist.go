package repository

import (
	"context"
	"errors"
	"wishlist-api/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WishlistRepo struct {
	pool *pgxpool.Pool
}

func NewWishlistRepo(pool *pgxpool.Pool) *WishlistRepo {
	return &WishlistRepo{pool: pool}
}

func (r *WishlistRepo) Create(ctx context.Context, w *domain.Wishlist) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO wishlists (user_id, title, description, event_date, share_token)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at, updated_at`,
		w.UserID, w.Title, w.Description, w.EventDate, w.ShareToken,
	).Scan(&w.ID, &w.CreatedAt, &w.UpdatedAt)
}

func (r *WishlistRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error) {
	var w domain.Wishlist
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, title, description, event_date, share_token, created_at, updated_at
		 FROM wishlists WHERE id = $1`, id,
	).Scan(&w.ID, &w.UserID, &w.Title, &w.Description, &w.EventDate, &w.ShareToken, &w.CreatedAt, &w.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrWishlistNotFound
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *WishlistRepo) ListByUserID(ctx context.Context, userID int64) ([]domain.Wishlist, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, title, description, event_date, share_token, created_at, updated_at
		 FROM wishlists WHERE user_id = $1
		 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.Wishlist, 0)
	for rows.Next() {
		var w domain.Wishlist
		if err := rows.Scan(&w.ID, &w.UserID, &w.Title, &w.Description, &w.EventDate, &w.ShareToken, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, w)
	}
	return list, rows.Err()
}

func (r *WishlistRepo) Update(ctx context.Context, w *domain.Wishlist) error {
	err := r.pool.QueryRow(ctx,
		`UPDATE wishlists SET title = $1, description = $2, event_date = $3, updated_at = NOW()
		 WHERE id = $4
		 RETURNING id, user_id, title, description, event_date, share_token, created_at, updated_at`,
		w.Title, w.Description, w.EventDate, w.ID,
	).Scan(&w.ID, &w.UserID, &w.Title, &w.Description, &w.EventDate, &w.ShareToken, &w.CreatedAt, &w.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrWishlistNotFound
	}
	return err
}

func (r *WishlistRepo) Patch(ctx context.Context, id uuid.UUID, changes domain.WishlistChanges) (*domain.Wishlist, error) {
	var w domain.Wishlist
	err := r.pool.QueryRow(ctx, `UPDATE wishlists SET title = COALESCE($2, title), description = COALESCE($3, description), event_date = COALESCE($4, event_date), updated_at = NOW() WHERE id = $1 RETURNING id, user_id, title, description, event_date, share_token, created_at, updated_at`, id, changes.Title, changes.Description, changes.EventDate).Scan(&w.ID, &w.UserID, &w.Title, &w.Description, &w.EventDate, &w.ShareToken, &w.CreatedAt, &w.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrWishlistNotFound
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *WishlistRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM wishlists WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrWishlistNotFound
	}
	return nil
}

func (r *WishlistRepo) GetByShareToken(ctx context.Context, token string) (*domain.Wishlist, error) {
	var w domain.Wishlist
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, title, description, event_date, share_token, created_at, updated_at
		 FROM wishlists WHERE share_token = $1`, token,
	).Scan(&w.ID, &w.UserID, &w.Title, &w.Description, &w.EventDate, &w.ShareToken, &w.CreatedAt, &w.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrWishlistNotFound
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}
