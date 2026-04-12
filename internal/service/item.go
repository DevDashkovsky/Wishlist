package service

import (
	"context"
	"errors"
	"wishlist-api/internal/domain"

	"github.com/google/uuid"
)

var (
	ErrItemNotFound    = errors.New("item not found")
	ErrAlreadyReserved = errors.New("item already reserved")
)

type ItemRepository interface {
	Create(ctx context.Context, item *domain.Item) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Item, error)
	Update(ctx context.Context, item *domain.Item) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type WishlistRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Wishlist, error)
}

type ItemService struct {
	items     ItemRepository
	wishlists WishlistRepository
}

func NewItemService(items ItemRepository, wishlists WishlistRepository) *ItemService {
	return &ItemService{items: items, wishlists: wishlists}
}

func (s *ItemService) Create(ctx context.Context, userID int64, wishlistID uuid.UUID, input domain.ItemInput) (*domain.Item, error) {
	if err := s.checkOwner(ctx, userID, wishlistID); err != nil {
		return nil, err
	}

	item := &domain.Item{
		WishlistID:  wishlistID,
		Title:       input.Title,
		Description: input.Description,
		URL:         input.URL,
		Priority:    input.Priority,
	}
	if err := s.items.Create(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *ItemService) Update(ctx context.Context, userID int64, wishlistID uuid.UUID, itemID uuid.UUID, input domain.ItemInput) (*domain.Item, error) {
	if err := s.checkOwner(ctx, userID, wishlistID); err != nil {
		return nil, err
	}

	item, err := s.items.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil || item.WishlistID != wishlistID {
		return nil, ErrItemNotFound
	}

	item.Title = input.Title
	item.Description = input.Description
	item.URL = input.URL
	item.Priority = input.Priority

	if err := s.items.Update(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *ItemService) Patch(ctx context.Context, userID int64, wishlistID uuid.UUID, itemID uuid.UUID, input domain.ItemPatch) (*domain.Item, error) {
	if err := s.checkOwner(ctx, userID, wishlistID); err != nil {
		return nil, err
	}

	item, err := s.items.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil || item.WishlistID != wishlistID {
		return nil, ErrItemNotFound
	}

	if input.Title != nil {
		item.Title = *input.Title
	}
	if input.Description != nil {
		item.Description = *input.Description
	}
	if input.URL != nil {
		item.URL = *input.URL
	}
	if input.Priority != nil {
		item.Priority = *input.Priority
	}

	if err := s.items.Update(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *ItemService) Delete(ctx context.Context, userID int64, wishlistID uuid.UUID, itemID uuid.UUID) error {
	if err := s.checkOwner(ctx, userID, wishlistID); err != nil {
		return err
	}

	item, err := s.items.GetByID(ctx, itemID)
	if err != nil {
		return err
	}
	if item == nil || item.WishlistID != wishlistID {
		return ErrItemNotFound
	}

	return s.items.Delete(ctx, itemID)
}

func (s *ItemService) checkOwner(ctx context.Context, userID int64, wishlistID uuid.UUID) error {
	w, err := s.wishlists.GetByID(ctx, wishlistID)
	if err != nil {
		return err
	}
	if w == nil {
		return ErrWishlistNotFound
	}
	if w.UserID != userID {
		return ErrForbidden
	}
	return nil
}
