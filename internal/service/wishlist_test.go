package service

import (
	"context"
	"errors"
	"testing"
	"wishlist-api/internal/domain"

	"github.com/google/uuid"
)

type mockWishlistFullRepo struct {
	create       func(ctx context.Context, w *domain.Wishlist) error
	getByID      func(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error)
	listByUserID func(ctx context.Context, userID int64) ([]domain.Wishlist, error)
	update       func(ctx context.Context, w *domain.Wishlist) error
	delete       func(ctx context.Context, id uuid.UUID) error
}

func (m *mockWishlistFullRepo) Create(ctx context.Context, w *domain.Wishlist) error {
	return m.create(ctx, w)
}
func (m *mockWishlistFullRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error) {
	return m.getByID(ctx, id)
}
func (m *mockWishlistFullRepo) ListByUserID(ctx context.Context, userID int64) ([]domain.Wishlist, error) {
	return m.listByUserID(ctx, userID)
}
func (m *mockWishlistFullRepo) Update(ctx context.Context, w *domain.Wishlist) error {
	return m.update(ctx, w)
}
func (m *mockWishlistFullRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.delete(ctx, id)
}

type mockWishlistItemRepo struct {
	listByWishlistID func(ctx context.Context, wishlistID uuid.UUID) ([]domain.Item, error)
}

func (m *mockWishlistItemRepo) ListByWishlistID(ctx context.Context, wishlistID uuid.UUID) ([]domain.Item, error) {
	return m.listByWishlistID(ctx, wishlistID)
}

var (
	wlTestUserID  int64 = 1
	wlTestID            = uuid.New()
	wlOtherUserID int64 = 99
)

func wlOwned() *domain.Wishlist {
	return &domain.Wishlist{ID: wlTestID, UserID: wlTestUserID, Title: "Birthday"}
}

func TestWishlistCreate_Success(t *testing.T) {
	svc := NewWishlistService(
		&mockWishlistFullRepo{
			create: func(ctx context.Context, w *domain.Wishlist) error {
				w.ID = wlTestID
				return nil
			},
		},
		&mockWishlistItemRepo{},
	)

	wl, err := svc.Create(context.Background(), wlTestUserID, domain.WishlistInput{
		Title:     "Birthday",
		EventDate: "2026-06-15",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wl.Title != "Birthday" {
		t.Errorf("got title %q, want %q", wl.Title, "Birthday")
	}
	if wl.ShareToken == "" {
		t.Error("expected non-empty share token")
	}
}

func TestWishlistCreate_InvalidDate(t *testing.T) {
	svc := NewWishlistService(
		&mockWishlistFullRepo{},
		&mockWishlistItemRepo{},
	)

	_, err := svc.Create(context.Background(), wlTestUserID, domain.WishlistInput{
		Title:     "Test",
		EventDate: "not-a-date",
	})

	if !errors.Is(err, ErrInvalidDate) {
		t.Errorf("got %v, want ErrInvalidDate", err)
	}
}

func TestWishlistGetByID_Success(t *testing.T) {
	svc := NewWishlistService(
		&mockWishlistFullRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error) {
				return wlOwned(), nil
			},
		},
		&mockWishlistItemRepo{
			listByWishlistID: func(ctx context.Context, wishlistID uuid.UUID) ([]domain.Item, error) {
				return []domain.Item{{Title: "Keyboard"}}, nil
			},
		},
	)

	result, err := svc.GetByID(context.Background(), wlTestUserID, wlTestID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Title != "Birthday" {
		t.Errorf("got title %q, want %q", result.Title, "Birthday")
	}
	if len(result.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(result.Items))
	}
}

func TestWishlistGetByID_NotFound(t *testing.T) {
	svc := NewWishlistService(
		&mockWishlistFullRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error) {
				return nil, nil
			},
		},
		&mockWishlistItemRepo{},
	)

	_, err := svc.GetByID(context.Background(), wlTestUserID, wlTestID)

	if !errors.Is(err, ErrWishlistNotFound) {
		t.Errorf("got %v, want ErrWishlistNotFound", err)
	}
}

func TestWishlistGetByID_Forbidden(t *testing.T) {
	svc := NewWishlistService(
		&mockWishlistFullRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error) {
				return wlOwned(), nil // владелец — wlTestUserID
			},
		},
		&mockWishlistItemRepo{},
	)

	_, err := svc.GetByID(context.Background(), wlOtherUserID, wlTestID)

	if !errors.Is(err, ErrForbidden) {
		t.Errorf("got %v, want ErrForbidden", err)
	}
}

func TestWishlistList_Success(t *testing.T) {
	svc := NewWishlistService(
		&mockWishlistFullRepo{
			listByUserID: func(ctx context.Context, userID int64) ([]domain.Wishlist, error) {
				return []domain.Wishlist{*wlOwned()}, nil
			},
		},
		&mockWishlistItemRepo{},
	)

	list, err := svc.List(context.Background(), wlTestUserID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("got %d wishlists, want 1", len(list))
	}
	if list[0].Title != "Birthday" {
		t.Errorf("got title %q, want %q", list[0].Title, "Birthday")
	}
}

func TestWishlistUpdate_Success(t *testing.T) {
	svc := NewWishlistService(
		&mockWishlistFullRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error) {
				return wlOwned(), nil
			},
			update: func(ctx context.Context, w *domain.Wishlist) error {
				return nil
			},
		},
		&mockWishlistItemRepo{},
	)

	wl, err := svc.Update(context.Background(), wlTestUserID, wlTestID, domain.WishlistInput{
		Title:     "New Year",
		EventDate: "2027-01-01",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wl.Title != "New Year" {
		t.Errorf("got title %q, want %q", wl.Title, "New Year")
	}
}

func TestWishlistUpdate_InvalidDate(t *testing.T) {
	svc := NewWishlistService(
		&mockWishlistFullRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error) {
				return wlOwned(), nil
			},
		},
		&mockWishlistItemRepo{},
	)

	_, err := svc.Update(context.Background(), wlTestUserID, wlTestID, domain.WishlistInput{
		Title:     "Test",
		EventDate: "bad-date",
	})

	if !errors.Is(err, ErrInvalidDate) {
		t.Errorf("got %v, want ErrInvalidDate", err)
	}
}

func TestWishlistUpdate_Forbidden(t *testing.T) {
	svc := NewWishlistService(
		&mockWishlistFullRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error) {
				return wlOwned(), nil
			},
		},
		&mockWishlistItemRepo{},
	)

	_, err := svc.Update(context.Background(), wlOtherUserID, wlTestID, domain.WishlistInput{
		Title:     "Hack",
		EventDate: "2026-01-01",
	})

	if !errors.Is(err, ErrForbidden) {
		t.Errorf("got %v, want ErrForbidden", err)
	}
}

func TestWishlistPatch_Success(t *testing.T) {
	svc := NewWishlistService(
		&mockWishlistFullRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error) {
				return wlOwned(), nil
			},
			update: func(ctx context.Context, w *domain.Wishlist) error {
				return nil
			},
		},
		&mockWishlistItemRepo{},
	)

	newTitle := "Patched"
	wl, err := svc.Patch(context.Background(), wlTestUserID, wlTestID, domain.WishlistPatch{
		Title: &newTitle,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wl.Title != "Patched" {
		t.Errorf("got title %q, want %q", wl.Title, "Patched")
	}
}

func TestWishlistPatch_InvalidDate(t *testing.T) {
	svc := NewWishlistService(
		&mockWishlistFullRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error) {
				return wlOwned(), nil
			},
		},
		&mockWishlistItemRepo{},
	)

	badDate := "nope"
	_, err := svc.Patch(context.Background(), wlTestUserID, wlTestID, domain.WishlistPatch{
		EventDate: &badDate,
	})

	if !errors.Is(err, ErrInvalidDate) {
		t.Errorf("got %v, want ErrInvalidDate", err)
	}
}

func TestWishlistDelete_Success(t *testing.T) {
	svc := NewWishlistService(
		&mockWishlistFullRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error) {
				return wlOwned(), nil
			},
			delete: func(ctx context.Context, id uuid.UUID) error {
				return nil
			},
		},
		&mockWishlistItemRepo{},
	)

	err := svc.Delete(context.Background(), wlTestUserID, wlTestID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWishlistDelete_Forbidden(t *testing.T) {
	svc := NewWishlistService(
		&mockWishlistFullRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error) {
				return wlOwned(), nil
			},
		},
		&mockWishlistItemRepo{},
	)

	err := svc.Delete(context.Background(), wlOtherUserID, wlTestID)

	if !errors.Is(err, ErrForbidden) {
		t.Errorf("got %v, want ErrForbidden", err)
	}
}
