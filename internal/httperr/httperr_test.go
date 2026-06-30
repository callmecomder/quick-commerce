package httperr

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"quickcommerce/internal/domain"
)

func TestFromDomainError_Mapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"product not found", domain.ErrProductNotFound, http.StatusNotFound, "product_not_found"},
		{"user not found", domain.ErrUserNotFound, http.StatusNotFound, "user_not_found"},
		{"user inactive", domain.ErrUserInactive, http.StatusConflict, "user_inactive"},
		{"out of stock", domain.ErrOutOfStock, http.StatusConflict, "out_of_stock"},
		{"amount mismatch", domain.ErrAmountMismatch, http.StatusBadRequest, "amount_mismatch"},
		{"missing idempotency key", domain.ErrMissingIdempotencyKey, http.StatusBadRequest, "missing_idempotency_key"},
		{"invalid quantity", domain.ErrInvalidQuantity, http.StatusBadRequest, "invalid_quantity"},
		{"missing required field", domain.ErrMissingRequiredField, http.StatusBadRequest, "missing_required_field"},
		{"unknown error", errors.New("something unexpected"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := FromDomainError(tt.err)
			if apiErr.StatusCode != tt.wantStatus {
				t.Errorf("status: want %d got %d", tt.wantStatus, apiErr.StatusCode)
			}
			if apiErr.Code != tt.wantCode {
				t.Errorf("code: want %s got %s", tt.wantCode, apiErr.Code)
			}
		})
	}
}

func TestFromDomainError_WrappedErrors(t *testing.T) {
	wrapped := fmt.Errorf("context: %w", domain.ErrOutOfStock)
	apiErr := FromDomainError(wrapped)
	if apiErr.StatusCode != http.StatusConflict {
		t.Errorf("wrapped error: want 409 got %d", apiErr.StatusCode)
	}
	if apiErr.Code != "out_of_stock" {
		t.Errorf("wrapped error code: want out_of_stock got %s", apiErr.Code)
	}
}

func TestFromDomainError_WrappedMissingField(t *testing.T) {
	wrapped := fmt.Errorf("%w: user_id", domain.ErrMissingRequiredField)
	apiErr := FromDomainError(wrapped)
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("want 400 got %d", apiErr.StatusCode)
	}
	if apiErr.Code != "missing_required_field" {
		t.Errorf("want missing_required_field got %s", apiErr.Code)
	}
}

func TestAPIError_Error(t *testing.T) {
	e := &APIError{StatusCode: 400, Code: "test", Message: "test message"}
	if e.Error() != "test message" {
		t.Errorf("want 'test message' got '%s'", e.Error())
	}
}

func TestFromDomainError_UnknownHidesMessage(t *testing.T) {
	apiErr := FromDomainError(errors.New("secret db details"))
	if apiErr.Message != "internal server error" {
		t.Errorf("unknown errors must not leak details, got: %s", apiErr.Message)
	}
}
