package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"
	"wishlist-api/internal/domain"
	"wishlist-api/internal/repository"

	"github.com/google/uuid"
)

const dateFormat = "2006-01-02"

var (
	ErrWishlistNotFound = errors.New("wishlist not found")
	ErrForbidden        = errors.New("forbidden")
	ErrInvalidDate      = errors.New("invalid date format, expected YYYY-MM-DD")
)

func parseDate(s string) (time.Time, error) {
	t, err := time.Parse(dateFormat, s)
	if err != nil {
		return time.Time{}, ErrInvalidDate
	}
	return t, nil
}

type WishlistService struct {
	wishlists *repository.WishlistRepo
	items     *repository.ItemRepo
}

func NewWishlistService(wishlists *repository.WishlistRepo, items *repository.ItemRepo) *WishlistService {
	return &WishlistService{wishlists: wishlists, items: items}
}

func (s *WishlistService) Create(ctx context.Context, userID int64, input domain.WishlistInput) (*domain.Wishlist, error) {
	eventDate, err := parseDate(input.EventDate)
	if err != nil {
		return nil, err
	}

	token, err := generateShareToken()
	if err != nil {
		return nil, err
	}

	w := &domain.Wishlist{
		UserID:      userID,
		Title:       input.Title,
		Description: input.Description,
		EventDate:   eventDate,
		ShareToken:  token,
	}
	if err := s.wishlists.Create(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *WishlistService) GetByID(ctx context.Context, userID int64, id uuid.UUID) (*domain.WishlistWithItems, error) {
	w, err := s.getOwnedWishlist(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	items, err := s.items.ListByWishlistID(ctx, w.ID)
	if err != nil {
		return nil, err
	}

	return &domain.WishlistWithItems{Wishlist: *w, Items: items}, nil
}

func (s *WishlistService) List(ctx context.Context, userID int64) ([]domain.Wishlist, error) {
	return s.wishlists.ListByUserID(ctx, userID)
}

func (s *WishlistService) Update(ctx context.Context, userID int64, id uuid.UUID, input domain.WishlistInput) (*domain.Wishlist, error) {
	w, err := s.getOwnedWishlist(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	eventDate, err := parseDate(input.EventDate)
	if err != nil {
		return nil, err
	}

	w.Title = input.Title
	w.Description = input.Description
	w.EventDate = eventDate

	if err := s.wishlists.Update(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *WishlistService) Patch(ctx context.Context, userID int64, id uuid.UUID, input domain.WishlistPatch) (*domain.Wishlist, error) {
	w, err := s.getOwnedWishlist(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	if input.Title != nil {
		w.Title = *input.Title
	}
	if input.Description != nil {
		w.Description = *input.Description
	}
	if input.EventDate != nil {
		eventDate, err := parseDate(*input.EventDate)
		if err != nil {
			return nil, err
		}
		w.EventDate = eventDate
	}

	if err := s.wishlists.Update(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *WishlistService) Delete(ctx context.Context, userID int64, id uuid.UUID) error {
	_, err := s.getOwnedWishlist(ctx, userID, id)
	if err != nil {
		return err
	}
	return s.wishlists.Delete(ctx, id)
}

func (s *WishlistService) getOwnedWishlist(ctx context.Context, userID int64, id uuid.UUID) (*domain.Wishlist, error) {
	w, err := s.wishlists.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, ErrWishlistNotFound
	}
	if w.UserID != userID {
		return nil, ErrForbidden
	}
	return w, nil
}

func generateShareToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
