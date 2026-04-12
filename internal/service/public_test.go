package service

import (
	"context"
	"errors"
	"testing"
	"wishlist-api/internal/domain"

	"github.com/google/uuid"
)

type mockPublicWishlistRepo struct {
	getByShareToken func(ctx context.Context, token string) (*domain.Wishlist, error)
}

func (m *mockPublicWishlistRepo) GetByShareToken(ctx context.Context, token string) (*domain.Wishlist, error) {
	return m.getByShareToken(ctx, token)
}

type mockPublicItemRepo struct {
	listByWishlistID func(ctx context.Context, wishlistID uuid.UUID) ([]domain.Item, error)
	getByID          func(ctx context.Context, id uuid.UUID) (*domain.Item, error)
	reserve          func(ctx context.Context, id uuid.UUID) (*domain.Item, error)
}

func (m *mockPublicItemRepo) ListByWishlistID(ctx context.Context, wishlistID uuid.UUID) ([]domain.Item, error) {
	return m.listByWishlistID(ctx, wishlistID)
}
func (m *mockPublicItemRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Item, error) {
	return m.getByID(ctx, id)
}
func (m *mockPublicItemRepo) Reserve(ctx context.Context, id uuid.UUID) (*domain.Item, error) {
	return m.reserve(ctx, id)
}

var (
	publicWishlistID = uuid.New()
	publicItemID     = uuid.New()
	shareToken       = "abc123"
)

func sharedWishlist() *domain.Wishlist {
	return &domain.Wishlist{ID: publicWishlistID, Title: "Birthday", ShareToken: shareToken}
}

func TestPublicGet_Success(t *testing.T) {
	svc := NewPublicService(
		&mockPublicWishlistRepo{
			getByShareToken: func(ctx context.Context, token string) (*domain.Wishlist, error) {
				return sharedWishlist(), nil
			},
		},
		&mockPublicItemRepo{
			listByWishlistID: func(ctx context.Context, wishlistID uuid.UUID) ([]domain.Item, error) {
				return []domain.Item{
					{ID: publicItemID, WishlistID: publicWishlistID, Title: "Keyboard"},
				}, nil
			},
		},
	)

	result, err := svc.GetByShareToken(context.Background(), shareToken)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Title != "Birthday" {
		t.Errorf("got title %q, want %q", result.Title, "Birthday")
	}
	if len(result.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(result.Items))
	}
	if result.Items[0].Title != "Keyboard" {
		t.Errorf("got item title %q, want %q", result.Items[0].Title, "Keyboard")
	}
}

func TestPublicGet_NotFound(t *testing.T) {
	svc := NewPublicService(
		&mockPublicWishlistRepo{
			getByShareToken: func(ctx context.Context, token string) (*domain.Wishlist, error) {
				return nil, nil
			},
		},
		&mockPublicItemRepo{},
	)

	_, err := svc.GetByShareToken(context.Background(), "nonexistent")

	if !errors.Is(err, ErrWishlistNotFound) {
		t.Errorf("got %v, want ErrWishlistNotFound", err)
	}
}

func TestPublicReserve_Success(t *testing.T) {
	svc := NewPublicService(
		&mockPublicWishlistRepo{
			getByShareToken: func(ctx context.Context, token string) (*domain.Wishlist, error) {
				return sharedWishlist(), nil
			},
		},
		&mockPublicItemRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Item, error) {
				return &domain.Item{ID: publicItemID, WishlistID: publicWishlistID, IsReserved: false}, nil
			},
			reserve: func(ctx context.Context, id uuid.UUID) (*domain.Item, error) {
				return &domain.Item{ID: publicItemID, WishlistID: publicWishlistID, IsReserved: true}, nil
			},
		},
	)

	item, err := svc.Reserve(context.Background(), shareToken, publicItemID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !item.IsReserved {
		t.Error("expected item to be reserved")
	}
}

func TestPublicReserve_AlreadyReserved(t *testing.T) {
	svc := NewPublicService(
		&mockPublicWishlistRepo{
			getByShareToken: func(ctx context.Context, token string) (*domain.Wishlist, error) {
				return sharedWishlist(), nil
			},
		},
		&mockPublicItemRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Item, error) {
				return &domain.Item{ID: publicItemID, WishlistID: publicWishlistID, IsReserved: true}, nil
			},
			reserve: func(ctx context.Context, id uuid.UUID) (*domain.Item, error) {
				return nil, nil // уже забронирован — returning nil
			},
		},
	)

	_, err := svc.Reserve(context.Background(), shareToken, publicItemID)

	if !errors.Is(err, ErrAlreadyReserved) {
		t.Errorf("got %v, want ErrAlreadyReserved", err)
	}
}

func TestPublicReserve_WishlistNotFound(t *testing.T) {
	svc := NewPublicService(
		&mockPublicWishlistRepo{
			getByShareToken: func(ctx context.Context, token string) (*domain.Wishlist, error) {
				return nil, nil
			},
		},
		&mockPublicItemRepo{},
	)

	_, err := svc.Reserve(context.Background(), "bad-token", publicItemID)

	if !errors.Is(err, ErrWishlistNotFound) {
		t.Errorf("got %v, want ErrWishlistNotFound", err)
	}
}

func TestPublicReserve_ItemNotFound(t *testing.T) {
	otherWishlistID := uuid.New()

	svc := NewPublicService(
		&mockPublicWishlistRepo{
			getByShareToken: func(ctx context.Context, token string) (*domain.Wishlist, error) {
				return sharedWishlist(), nil
			},
		},
		&mockPublicItemRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Item, error) {
				// айтем существует, но принадлежит другому вишлисту
				return &domain.Item{ID: id, WishlistID: otherWishlistID}, nil
			},
		},
	)

	_, err := svc.Reserve(context.Background(), shareToken, publicItemID)

	if !errors.Is(err, ErrItemNotFound) {
		t.Errorf("got %v, want ErrItemNotFound", err)
	}
}
