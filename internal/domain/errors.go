package domain

import "errors"

var (
	ErrProductNotFound       = errors.New("product not found")
	ErrUserNotFound          = errors.New("user not found")
	ErrUserInactive          = errors.New("user is not active")
	ErrOutOfStock            = errors.New("product out of stock")
	ErrAmountMismatch        = errors.New("amount does not match product price")
	ErrMissingIdempotencyKey = errors.New("Idempotency-Key header is required")
	ErrPaymentFailed         = errors.New("payment failed")
	ErrInvalidQuantity       = errors.New("quantity must be greater than 0")
	ErrMissingRequiredField  = errors.New("missing required field")
)
