package service

import (
	"fmt"

	"quickcommerce/internal/domain"
)

// validatePlaceOrderRequest checks all mandatory fields on the order request.
func validatePlaceOrderRequest(req PlaceOrderRequest, idempotencyKey string) error {
	if idempotencyKey == "" {
		return domain.ErrMissingIdempotencyKey
	}
	if req.UserID == "" {
		return fmt.Errorf("%w: user_id", domain.ErrMissingRequiredField)
	}
	if req.ProductID == "" {
		return fmt.Errorf("%w: product_id", domain.ErrMissingRequiredField)
	}
	if req.Quantity <= 0 {
		return domain.ErrInvalidQuantity
	}
	if req.Amount <= 0 {
		return fmt.Errorf("%w: amount", domain.ErrMissingRequiredField)
	}
	if req.Latitude == "" {
		return fmt.Errorf("%w: latitude", domain.ErrMissingRequiredField)
	}
	if req.Longitude == "" {
		return fmt.Errorf("%w: longitude", domain.ErrMissingRequiredField)
	}
	if len(req.Metadata) == 0 {
		return fmt.Errorf("%w: metadata", domain.ErrMissingRequiredField)
	}
	if req.Metadata["address"] == "" {
		return fmt.Errorf("%w: metadata.address", domain.ErrMissingRequiredField)
	}
	return nil
}
