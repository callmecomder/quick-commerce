package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(ph *ProductHandler, oh *OrderHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Route("/v1", func(r chi.Router) {
		r.Get("/products", ph.Search)
		r.Get("/products/{product_id}", ph.GetByID)
		r.Post("/orders", oh.PlaceOrder)
	})

	return r
}
