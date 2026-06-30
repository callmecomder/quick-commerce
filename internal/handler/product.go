package handler

import (
	"encoding/json"
	"net/http"

	"quickcommerce/internal/httperr"
	"quickcommerce/internal/service"

	"github.com/go-chi/chi/v5"
)

type ProductHandler struct {
	svc *service.ProductService
}

func NewProductHandler(svc *service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

func (h *ProductHandler) Search(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("product_name")

	products, err := h.svc.Search(r.Context(), name)
	if err != nil {
		apiErr := httperr.FromDomainError(err)
		writeJSON(w, apiErr.StatusCode, apiErr)
		return
	}

	resp := SearchProductsResponse{Products: make([]ProductDTO, 0, len(products))}
	for _, p := range products {
		resp.Products = append(resp.Products, ProductDTO{
			ProductID:   p.ID,
			Description: p.Description,
			Brand:       p.Brand,
			Quantity:    p.Quantity,
			Amount:      p.Amount,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *ProductHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "product_id")

	product, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		apiErr := httperr.FromDomainError(err)
		writeJSON(w, apiErr.StatusCode, apiErr)
		return
	}

	writeJSON(w, http.StatusOK, ProductDTO{
		ProductID:   product.ID,
		Description: product.Description,
		Brand:       product.Brand,
		Quantity:    product.Quantity,
		Amount:      product.Amount,
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
