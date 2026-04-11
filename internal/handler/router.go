package handler

import (
	"wishlist-api/internal/jwt"
	"wishlist-api/internal/middleware"

	"github.com/go-chi/chi/v5"
)

func NewRouter(
	jwtManager *jwt.Manager,
	auth *AuthHandler,
	wishlists *WishlistHandler,
	items *ItemHandler,
	public *PublicHandler,
) *chi.Mux {
	r := chi.NewRouter()

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
