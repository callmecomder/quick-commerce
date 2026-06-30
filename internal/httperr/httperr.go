package httperr

import (
	"errors"
	"net/http"

	"quickcommerce/internal/domain"
)

type APIError struct {
	StatusCode int    `json:"-"`
	Code       string `json:"error"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	return e.Message
}

func FromDomainError(err error) *APIError {
	switch {
	case errors.Is(err, domain.ErrProductNotFound):
		return &APIError{StatusCode: http.StatusNotFound, Code: "product_not_found", Message: err.Error()}
	case errors.Is(err, domain.ErrUserNotFound):
		return &APIError{StatusCode: http.StatusNotFound, Code: "user_not_found", Message: err.Error()}
	case errors.Is(err, domain.ErrUserInactive):
		return &APIError{StatusCode: http.StatusConflict, Code: "user_inactive", Message: err.Error()}
	case errors.Is(err, domain.ErrOutOfStock):
		return &APIError{StatusCode: http.StatusConflict, Code: "out_of_stock", Message: err.Error()}
	case errors.Is(err, domain.ErrAmountMismatch):
		return &APIError{StatusCode: http.StatusBadRequest, Code: "amount_mismatch", Message: err.Error()}
	case errors.Is(err, domain.ErrMissingIdempotencyKey):
		return &APIError{StatusCode: http.StatusBadRequest, Code: "missing_idempotency_key", Message: err.Error()}
	case errors.Is(err, domain.ErrInvalidQuantity):
		return &APIError{StatusCode: http.StatusBadRequest, Code: "invalid_quantity", Message: err.Error()}
	case errors.Is(err, domain.ErrMissingRequiredField):
		return &APIError{StatusCode: http.StatusBadRequest, Code: "missing_required_field", Message: err.Error()}
	default:
		return &APIError{StatusCode: http.StatusInternalServerError, Code: "internal_error", Message: "internal server error"}
	}
}
