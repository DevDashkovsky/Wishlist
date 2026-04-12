package service

import (
	"context"
	"wishlist-api/internal/domain"

	"github.com/google/uuid"
)

type PublicWishlistRepository interface {
	GetByShareToken(ctx context.Context, token string) (*domain.Wishlist, error)
}

type PublicItemRepository interface {
	ListByWishlistID(ctx context.Context, wishlistID uuid.UUID) ([]domain.Item, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Item, error)
	Reserve(ctx context.Context, id uuid.UUID) (*domain.Item, error)
}

type PublicService struct {
	wishlists PublicWishlistRepository
	items     PublicItemRepository
}

func NewPublicService(wishlists PublicWishlistRepository, items PublicItemRepository) *PublicService {
	return &PublicService{wishlists: wishlists, items: items}
}

func (s *PublicService) GetByShareToken(ctx context.Context, token string) (*domain.PublicWishlistWithItems, error) {
	w, err := s.wishlists.GetByShareToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, ErrWishlistNotFound
	}

	items, err := s.items.ListByWishlistID(ctx, w.ID)
	if err != nil {
		return nil, err
	}

	pub := &domain.PublicWishlistWithItems{
		PublicWishlist: domain.PublicWishlist{
			ID:          w.ID,
			Title:       w.Title,
			Description: w.Description,
			EventDate:   w.EventDate,
			CreatedAt:   w.CreatedAt,
		},
		Items: items,
	}
	return pub, nil
}

func (s *PublicService) Reserve(ctx context.Context, token string, itemID uuid.UUID) (*domain.Item, error) {
	w, err := s.wishlists.GetByShareToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, ErrWishlistNotFound
	}

	item, err := s.items.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil || item.WishlistID != w.ID {
		return nil, ErrItemNotFound
	}

	reserved, err := s.items.Reserve(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if reserved == nil {
		return nil, ErrAlreadyReserved
	}
	return reserved, nil
}
