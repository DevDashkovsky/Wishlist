package service

import (
	"context"
	"errors"
	"testing"
	"wishlist-api/internal/domain"

	"github.com/google/uuid"
)

type mockItemRepo struct {
	patch   func(context.Context, uuid.UUID, domain.ItemPatch) (*domain.Item, error)
	create  func(ctx context.Context, item *domain.Item) error
	getByID func(ctx context.Context, id uuid.UUID) (*domain.Item, error)
	update  func(ctx context.Context, item *domain.Item) error
	delete  func(ctx context.Context, id uuid.UUID) error
}

func (m *mockItemRepo) Create(ctx context.Context, item *domain.Item) error {
	return m.create(ctx, item)
}
func (m *mockItemRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Item, error) {
	return m.getByID(ctx, id)
}
func (m *mockItemRepo) Update(ctx context.Context, item *domain.Item) error {
	return m.update(ctx, item)
}
func (m *mockItemRepo) Patch(ctx context.Context, id uuid.UUID, input domain.ItemPatch) (*domain.Item, error) {
	return m.patch(ctx, id, input)
}
func (m *mockItemRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.delete(ctx, id)
}

type mockWishlistRepo struct {
	getByID func(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error)
}

func (m *mockWishlistRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error) {
	return m.getByID(ctx, id)
}

var (
	testUserID     int64 = 1
	testWishlistID       = uuid.New()
	testItemID           = uuid.New()
	otherUserID    int64 = 99
)

func ownedWishlist() *domain.Wishlist {
	return &domain.Wishlist{ID: testWishlistID, UserID: testUserID}
}

func testItem() *domain.Item {
	return &domain.Item{
		ID:         testItemID,
		WishlistID: testWishlistID,
		Title:      "Keyboard",
		Priority:   3,
	}
}

func TestItemCreate_Success(t *testing.T) {
	svc := NewItemService(
		&mockItemRepo{
			create: func(ctx context.Context, item *domain.Item) error {
				item.ID = testItemID
				return nil
			},
		},
		&mockWishlistRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error) {
				return ownedWishlist(), nil
			},
		},
	)

	item, err := svc.Create(context.Background(), testUserID, testWishlistID, domain.ItemInput{
		Title:    "Keyboard",
		Priority: 3,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Title != "Keyboard" {
		t.Errorf("got title %q, want %q", item.Title, "Keyboard")
	}
}

func TestItemCreate_WishlistNotFound(t *testing.T) {
	svc := NewItemService(
		&mockItemRepo{},
		&mockWishlistRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error) {
				return nil, nil
			},
		},
	)

	_, err := svc.Create(context.Background(), testUserID, testWishlistID, domain.ItemInput{
		Title:    "Keyboard",
		Priority: 3,
	})

	if !errors.Is(err, ErrWishlistNotFound) {
		t.Errorf("got %v, want ErrWishlistNotFound", err)
	}
}

func TestItemCreate_Forbidden(t *testing.T) {
	svc := NewItemService(
		&mockItemRepo{},
		&mockWishlistRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error) {
				return ownedWishlist(), nil
			},
		},
	)

	_, err := svc.Create(context.Background(), otherUserID, testWishlistID, domain.ItemInput{
		Title:    "Keyboard",
		Priority: 3,
	})

	if !errors.Is(err, ErrForbidden) {
		t.Errorf("got %v, want ErrForbidden", err)
	}
}

func TestItemUpdate_Success(t *testing.T) {
	svc := NewItemService(
		&mockItemRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Item, error) {
				return testItem(), nil
			},
			update: func(ctx context.Context, item *domain.Item) error {
				return nil
			},
		},
		&mockWishlistRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error) {
				return ownedWishlist(), nil
			},
		},
	)

	item, err := svc.Update(context.Background(), testUserID, testWishlistID, testItemID, domain.ItemInput{
		Title:    "Updated keyboard",
		Priority: 5,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Title != "Updated keyboard" {
		t.Errorf("got title %q, want %q", item.Title, "Updated keyboard")
	}
	if item.Priority != 5 {
		t.Errorf("got priority %d, want %d", item.Priority, 5)
	}
}

func TestItemUpdate_ItemNotFound(t *testing.T) {
	svc := NewItemService(
		&mockItemRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Item, error) {
				return nil, nil
			},
		},
		&mockWishlistRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error) {
				return ownedWishlist(), nil
			},
		},
	)

	_, err := svc.Update(context.Background(), testUserID, testWishlistID, testItemID, domain.ItemInput{
		Title:    "X",
		Priority: 3,
	})

	if !errors.Is(err, ErrItemNotFound) {
		t.Errorf("got %v, want ErrItemNotFound", err)
	}
}

func TestItemPatch_Success(t *testing.T) {
	svc := NewItemService(
		&mockItemRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Item, error) {
				return testItem(), nil
			},
			patch: func(ctx context.Context, id uuid.UUID, changes domain.ItemPatch) (*domain.Item, error) {
				if changes.Description != nil || changes.URL != nil {
					t.Fatal("patch forwarded unchanged fields")
				}
				item := testItem()
				if changes.Title != nil {
					item.Title = *changes.Title
				}
				if changes.Priority != nil {
					item.Priority = *changes.Priority
				}
				return item, nil
			},
		},
		&mockWishlistRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error) {
				return ownedWishlist(), nil
			},
		},
	)

	newPriority := 1
	item, err := svc.Patch(context.Background(), testUserID, testWishlistID, testItemID, domain.ItemPatch{
		Priority: &newPriority,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Priority != 1 {
		t.Errorf("got priority %d, want %d", item.Priority, 1)
	}

	if item.Title != "Keyboard" {
		t.Errorf("got title %q, want %q", item.Title, "Keyboard")
	}
}

func TestItemDelete_Success(t *testing.T) {
	svc := NewItemService(
		&mockItemRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Item, error) {
				return testItem(), nil
			},
			delete: func(ctx context.Context, id uuid.UUID) error {
				return nil
			},
		},
		&mockWishlistRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error) {
				return ownedWishlist(), nil
			},
		},
	)

	err := svc.Delete(context.Background(), testUserID, testWishlistID, testItemID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestItemDelete_Forbidden(t *testing.T) {
	svc := NewItemService(
		&mockItemRepo{},
		&mockWishlistRepo{
			getByID: func(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error) {
				return ownedWishlist(), nil
			},
		},
	)

	err := svc.Delete(context.Background(), otherUserID, testWishlistID, testItemID)

	if !errors.Is(err, ErrForbidden) {
		t.Errorf("got %v, want ErrForbidden", err)
	}
}
