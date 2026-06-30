package handler

import (
	"encoding/json"
	"net/http"

	"quickcommerce/internal/httperr"
	"quickcommerce/internal/service"
)

type OrderHandler struct {
	svc *service.OrderService
}

func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

func (h *OrderHandler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := r.Header.Get("Idempotency-Key")

	var req PlaceOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "invalid request body",
		})
		return
	}

	result, err := h.svc.PlaceOrder(r.Context(), service.PlaceOrderRequest{
		UserID:    req.UserID,
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
		Amount:    req.Amount,
		Metadata:  req.Metadata,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
	}, idempotencyKey)
	if err != nil {
		apiErr := httperr.FromDomainError(err)
		writeJSON(w, apiErr.StatusCode, apiErr)
		return
	}

	writeJSON(w, http.StatusOK, OrderResponse{
		ID:                 result.Order.ID,
		UserID:             result.Order.UserID,
		Status:             string(result.Order.Status),
		ProductID:          result.Order.ProductID,
		ProductName:        result.ProductBrand,
		ProductDescription: result.ProductDescription,
		Quantity:           result.Order.Quantity,
		Amount:             result.Order.Amount,
		FailureReason:      result.Order.FailureReason,
		CreatedAt:          result.Order.CreatedAt,
	})
}
