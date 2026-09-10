package handler

import (
	"net/http"
	"net/url"
	"strings"
	"wishlist-api/internal/domain"
	"wishlist-api/internal/middleware"
	"wishlist-api/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ItemHandler struct {
	items *service.ItemService
}

func NewItemHandler(items *service.ItemService) *ItemHandler {
	return &ItemHandler{items: items}
}

func (h *ItemHandler) Create(w http.ResponseWriter, r *http.Request) {
	wishlistID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "wishlist not found")
		return
	}

	input := domain.ItemInput{Priority: 3}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	if strings.TrimSpace(input.Title) == "" {
		writeError(w, http.StatusUnprocessableEntity, "title is required")
		return
	}
	if input.Priority < 1 || input.Priority > 5 {
		writeError(w, http.StatusUnprocessableEntity, "priority must be between 1 and 5")
		return
	}

	if !validURL(input.URL) {
		writeError(w, http.StatusUnprocessableEntity, "url must be an absolute URI or empty")
		return
	}

	userID := middleware.UserID(r.Context())
	item, err := h.items.Create(r.Context(), userID, wishlistID, input)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, item)
}

func (h *ItemHandler) Update(w http.ResponseWriter, r *http.Request) {
	wishlistID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "wishlist not found")
		return
	}
	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		writeError(w, http.StatusNotFound, "item not found")
		return
	}

	input := domain.ItemInput{Priority: 3}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	if strings.TrimSpace(input.Title) == "" {
		writeError(w, http.StatusUnprocessableEntity, "title is required")
		return
	}
	if input.Priority < 1 || input.Priority > 5 {
		writeError(w, http.StatusUnprocessableEntity, "priority must be between 1 and 5")
		return
	}

	if !validURL(input.URL) {
		writeError(w, http.StatusUnprocessableEntity, "url must be an absolute URI or empty")
		return
	}

	userID := middleware.UserID(r.Context())
	item, err := h.items.Update(r.Context(), userID, wishlistID, itemID, input)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *ItemHandler) Patch(w http.ResponseWriter, r *http.Request) {
	wishlistID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "wishlist not found")
		return
	}
	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		writeError(w, http.StatusNotFound, "item not found")
		return
	}

	var input domain.ItemPatch
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	if input.Priority != nil && (*input.Priority < 1 || *input.Priority > 5) {
		writeError(w, http.StatusUnprocessableEntity, "priority must be between 1 and 5")
		return
	}

	if input.Title != nil && strings.TrimSpace(*input.Title) == "" {
		writeError(w, http.StatusUnprocessableEntity, "title is required")
		return
	}

	if input.URL != nil && !validURL(*input.URL) {
		writeError(w, http.StatusUnprocessableEntity, "url must be an absolute URI or empty")
		return
	}

	userID := middleware.UserID(r.Context())
	item, err := h.items.Patch(r.Context(), userID, wishlistID, itemID, input)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *ItemHandler) Delete(w http.ResponseWriter, r *http.Request) {
	wishlistID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "wishlist not found")
		return
	}
	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		writeError(w, http.StatusNotFound, "item not found")
		return
	}

	userID := middleware.UserID(r.Context())
	if err := h.items.Delete(r.Context(), userID, wishlistID, itemID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func validURL(value string) bool {
	if value == "" {
		return true
	}
	parsed, err := url.ParseRequestURI(value)
	return err == nil && parsed.IsAbs() && ((parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() != "")
}
