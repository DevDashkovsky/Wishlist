package handler

import (
	"net/http"
	"wishlist-api/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type PublicHandler struct {
	public *service.PublicService
}

func NewPublicHandler(public *service.PublicService) *PublicHandler {
	return &PublicHandler{public: public}
}

func (h *PublicHandler) GetByShareToken(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		writeError(w, http.StatusNotFound, "wishlist not found")
		return
	}

	wl, err := h.public.GetByShareToken(r.Context(), token)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, wl)
}

func (h *PublicHandler) Reserve(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		writeError(w, http.StatusNotFound, "wishlist not found")
		return
	}

	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		writeError(w, http.StatusNotFound, "item not found")
		return
	}

	item, err := h.public.Reserve(r.Context(), token, itemID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, item)
}
