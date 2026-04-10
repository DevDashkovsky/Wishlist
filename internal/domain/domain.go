package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Wishlist struct {
	ID          uuid.UUID `json:"id"`
	UserID      int64     `json:"-"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	EventDate   time.Time `json:"event_date"`
	ShareToken  string    `json:"share_token"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type WishlistWithItems struct {
	Wishlist
	Items []Item `json:"items"`
}

type PublicWishlist struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	EventDate   time.Time `json:"event_date"`
	CreatedAt   time.Time `json:"created_at"`
}

type PublicWishlistWithItems struct {
	PublicWishlist
	Items []Item `json:"items"`
}

type Item struct {
	ID          uuid.UUID `json:"id"`
	WishlistID  uuid.UUID `json:"wishlist_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	Priority    int       `json:"priority"`
	IsReserved  bool      `json:"is_reserved"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RegisterInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type WishlistInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	EventDate   string `json:"event_date"`
}

type WishlistPatch struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	EventDate   *string `json:"event_date"`
}

type ItemInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Priority    int    `json:"priority"`
}

type ItemPatch struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	URL         *string `json:"url"`
	Priority    *int    `json:"priority"`
}
