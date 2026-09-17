package handler

import (
	"context"
	"net/http"
	"time"

	"wishlist-api/internal/jwt"
	"wishlist-api/internal/middleware"

	"github.com/go-chi/chi/v5"
)

func NewRouter(
	requestTimeout time.Duration,
	readiness func(context.Context) error,
	jwtManager *jwt.Manager,
	auth *AuthHandler,
	wishlists *WishlistHandler,
	items *ItemHandler,
	public *PublicHandler,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Timeout(requestTimeout))
	r.Use(Recoverer)
	r.Use(LimitBody)
	r.NotFound(func(w http.ResponseWriter, _ *http.Request) { writeError(w, http.StatusNotFound, "not found") })
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	})

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := readiness(r.Context()); err != nil {
			writeError(w, http.StatusServiceUnavailable, "service unavailable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", auth.Register)
		r.Post("/auth/login", auth.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(jwtManager))

			r.Get("/wishlists", wishlists.List)
			r.Post("/wishlists", wishlists.Create)
			r.Get("/wishlists/{id}", wishlists.Get)
			r.Put("/wishlists/{id}", wishlists.Update)
			r.Patch("/wishlists/{id}", wishlists.Patch)
			r.Delete("/wishlists/{id}", wishlists.Delete)

			r.Post("/wishlists/{id}/items", items.Create)
			r.Put("/wishlists/{id}/items/{itemId}", items.Update)
			r.Patch("/wishlists/{id}/items/{itemId}", items.Patch)
			r.Delete("/wishlists/{id}/items/{itemId}", items.Delete)
		})

		r.Get("/shared/{token}", public.GetByShareToken)
		r.Post("/shared/{token}/items/{itemId}/reserve", public.Reserve)
	})

	return r
}
