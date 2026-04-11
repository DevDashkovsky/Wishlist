package handler

import (
	"errors"
	"net/http"
	"wishlist-api/internal/domain"
	"wishlist-api/internal/middleware"
	"wishlist-api/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type WishlistHandler struct {
	wishlists *service.WishlistService
}

func NewWishlistHandler(wishlists *service.WishlistService) *WishlistHandler {
	return &WishlistHandler{wishlists: wishlists}
}

func (h *WishlistHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input domain.WishlistInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	if input.Title == "" {
		writeError(w, http.StatusUnprocessableEntity, "title is required")
		return
	}
	if input.EventDate == "" {
		writeError(w, http.StatusUnprocessableEntity, "event_date is required")
		return
	}

	userID := middleware.UserID(r.Context())
	wl, err := h.wishlists.Create(r.Context(), userID, input)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, wl)
}

func (h *WishlistHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	list, err := h.wishlists.List(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	if list == nil {
		list = []domain.Wishlist{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *WishlistHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "wishlist not found")
		return
	}

	userID := middleware.UserID(r.Context())
	wl, err := h.wishlists.GetByID(r.Context(), userID, id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, wl)
}

func (h *WishlistHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "wishlist not found")
		return
	}

	var input domain.WishlistInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	if input.Title == "" {
		writeError(w, http.StatusUnprocessableEntity, "title is required")
		return
	}
	if input.EventDate == "" {
		writeError(w, http.StatusUnprocessableEntity, "event_date is required")
		return
	}

	userID := middleware.UserID(r.Context())
	wl, err := h.wishlists.Update(r.Context(), userID, id, input)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, wl)
}

func (h *WishlistHandler) Patch(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "wishlist not found")
		return
	}

	var input domain.WishlistPatch
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	userID := middleware.UserID(r.Context())
	wl, err := h.wishlists.Patch(r.Context(), userID, id, input)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, wl)
}

func (h *WishlistHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "wishlist not found")
		return
	}

	userID := middleware.UserID(r.Context())
	if err := h.wishlists.Delete(r.Context(), userID, id); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrWishlistNotFound):
		writeError(w, http.StatusNotFound, "wishlist not found")
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, service.ErrItemNotFound):
		writeError(w, http.StatusNotFound, "item not found")
	case errors.Is(err, service.ErrAlreadyReserved):
		writeError(w, http.StatusConflict, "item already reserved")
	case errors.Is(err, service.ErrInvalidDate):
		writeError(w, http.StatusUnprocessableEntity, "invalid date format, expected YYYY-MM-DD")
	default:
		writeError(w, http.StatusInternalServerError, "something went wrong")
	}
}
