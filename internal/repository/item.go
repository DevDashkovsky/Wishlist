package repository

import (
	"context"
	"errors"
	"wishlist-api/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ItemRepo struct {
	pool *pgxpool.Pool
}

func NewItemRepo(pool *pgxpool.Pool) *ItemRepo {
	return &ItemRepo{pool: pool}
}

func (r *ItemRepo) Create(ctx context.Context, item *domain.Item) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO items (wishlist_id, title, description, url, priority)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, is_reserved, created_at, updated_at`,
		item.WishlistID, item.Title, item.Description, item.URL, item.Priority,
	).Scan(&item.ID, &item.IsReserved, &item.CreatedAt, &item.UpdatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return domain.ErrWishlistNotFound
	}
	return err
}

func (r *ItemRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Item, error) {
	var item domain.Item
	err := r.pool.QueryRow(ctx,
		`SELECT id, wishlist_id, title, description, url, priority, is_reserved, created_at, updated_at
		 FROM items WHERE id = $1`, id,
	).Scan(&item.ID, &item.WishlistID, &item.Title, &item.Description, &item.URL, &item.Priority, &item.IsReserved, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrItemNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ItemRepo) ListByWishlistID(ctx context.Context, wishlistID uuid.UUID) ([]domain.Item, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, wishlist_id, title, description, url, priority, is_reserved, created_at, updated_at
		 FROM items WHERE wishlist_id = $1
		 ORDER BY priority DESC, created_at`, wishlistID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]domain.Item, 0)
	for rows.Next() {
		var item domain.Item
		if err := rows.Scan(&item.ID, &item.WishlistID, &item.Title, &item.Description, &item.URL, &item.Priority, &item.IsReserved, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

func (r *ItemRepo) Update(ctx context.Context, item *domain.Item) error {
	err := r.pool.QueryRow(ctx,
		`UPDATE items SET title = $1, description = $2, url = $3, priority = $4, updated_at = NOW()
		 WHERE id = $5
		 RETURNING id, wishlist_id, title, description, url, priority, is_reserved, created_at, updated_at`,
		item.Title, item.Description, item.URL, item.Priority, item.ID,
	).Scan(&item.ID, &item.WishlistID, &item.Title, &item.Description, &item.URL, &item.Priority, &item.IsReserved, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrItemNotFound
	}
	return err
}

func (r *ItemRepo) Patch(ctx context.Context, id uuid.UUID, changes domain.ItemPatch) (*domain.Item, error) {
	var item domain.Item
	err := r.pool.QueryRow(ctx, `UPDATE items SET title = COALESCE($2, title), description = COALESCE($3, description), url = COALESCE($4, url), priority = COALESCE($5, priority), updated_at = NOW() WHERE id = $1 RETURNING id, wishlist_id, title, description, url, priority, is_reserved, created_at, updated_at`, id, changes.Title, changes.Description, changes.URL, changes.Priority).Scan(&item.ID, &item.WishlistID, &item.Title, &item.Description, &item.URL, &item.Priority, &item.IsReserved, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrItemNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ItemRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM items WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrItemNotFound
	}
	return nil
}

func (r *ItemRepo) Reserve(ctx context.Context, id uuid.UUID) (*domain.Item, error) {
	var item domain.Item
	err := r.pool.QueryRow(ctx,
		`UPDATE items SET is_reserved = TRUE, updated_at = NOW()
		 WHERE id = $1 AND is_reserved = FALSE
		 RETURNING id, wishlist_id, title, description, url, priority, is_reserved, created_at, updated_at`,
		id,
	).Scan(&item.ID, &item.WishlistID, &item.Title, &item.Description, &item.URL, &item.Priority, &item.IsReserved, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		_, lookupErr := r.GetByID(ctx, id)
		if lookupErr != nil {
			return nil, lookupErr
		}
		return nil, domain.ErrAlreadyReserved
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}
