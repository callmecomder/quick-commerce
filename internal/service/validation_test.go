package service

import (
	"errors"
	"testing"

	"quickcommerce/internal/domain"
)

func TestValidatePlaceOrderRequest_AllValid(t *testing.T) {
	req := PlaceOrderRequest{
		UserID:    "user-001",
		ProductID: "prod-001",
		Quantity:  1,
		Amount:    2000,
		Metadata:  map[string]string{"address": "123 Main St"},
		Latitude:  "12.97",
		Longitude: "77.59",
	}
	if err := validatePlaceOrderRequest(req, "key-1"); err != nil {
		t.Errorf("valid request returned error: %v", err)
	}
}

func TestValidatePlaceOrderRequest_MissingIdempotencyKey(t *testing.T) {
	req := PlaceOrderRequest{
		UserID:    "user-001",
		ProductID: "prod-001",
		Quantity:  1,
		Amount:    2000,
		Metadata:  map[string]string{"address": "123 Main St"},
		Latitude:  "12.97",
		Longitude: "77.59",
	}
	err := validatePlaceOrderRequest(req, "")
	if !errors.Is(err, domain.ErrMissingIdempotencyKey) {
		t.Errorf("want ErrMissingIdempotencyKey, got %v", err)
	}
}

func TestValidatePlaceOrderRequest_MissingFields(t *testing.T) {
	base := PlaceOrderRequest{
		UserID:    "user-001",
		ProductID: "prod-001",
		Quantity:  1,
		Amount:    2000,
		Metadata:  map[string]string{"address": "123 Main St"},
		Latitude:  "12.97",
		Longitude: "77.59",
	}

	tests := []struct {
		name    string
		modify  func(PlaceOrderRequest) PlaceOrderRequest
		wantErr error
	}{
		{
			name:    "empty user_id",
			modify:  func(r PlaceOrderRequest) PlaceOrderRequest { r.UserID = ""; return r },
			wantErr: domain.ErrMissingRequiredField,
		},
		{
			name:    "empty product_id",
			modify:  func(r PlaceOrderRequest) PlaceOrderRequest { r.ProductID = ""; return r },
			wantErr: domain.ErrMissingRequiredField,
		},
		{
			name:    "zero quantity",
			modify:  func(r PlaceOrderRequest) PlaceOrderRequest { r.Quantity = 0; return r },
			wantErr: domain.ErrInvalidQuantity,
		},
		{
			name:    "negative quantity",
			modify:  func(r PlaceOrderRequest) PlaceOrderRequest { r.Quantity = -1; return r },
			wantErr: domain.ErrInvalidQuantity,
		},
		{
			name:    "zero amount",
			modify:  func(r PlaceOrderRequest) PlaceOrderRequest { r.Amount = 0; return r },
			wantErr: domain.ErrMissingRequiredField,
		},
		{
			name:    "negative amount",
			modify:  func(r PlaceOrderRequest) PlaceOrderRequest { r.Amount = -100; return r },
			wantErr: domain.ErrMissingRequiredField,
		},
		{
			name:    "empty latitude",
			modify:  func(r PlaceOrderRequest) PlaceOrderRequest { r.Latitude = ""; return r },
			wantErr: domain.ErrMissingRequiredField,
		},
		{
			name:    "empty longitude",
			modify:  func(r PlaceOrderRequest) PlaceOrderRequest { r.Longitude = ""; return r },
			wantErr: domain.ErrMissingRequiredField,
		},
		{
			name:    "nil metadata",
			modify:  func(r PlaceOrderRequest) PlaceOrderRequest { r.Metadata = nil; return r },
			wantErr: domain.ErrMissingRequiredField,
		},
		{
			name:    "empty metadata map",
			modify:  func(r PlaceOrderRequest) PlaceOrderRequest { r.Metadata = map[string]string{}; return r },
			wantErr: domain.ErrMissingRequiredField,
		},
		{
			name: "metadata without address",
			modify: func(r PlaceOrderRequest) PlaceOrderRequest {
				r.Metadata = map[string]string{"note": "hello"}
				return r
			},
			wantErr: domain.ErrMissingRequiredField,
		},
		{
			name: "metadata with empty address",
			modify: func(r PlaceOrderRequest) PlaceOrderRequest {
				r.Metadata = map[string]string{"address": ""}
				return r
			},
			wantErr: domain.ErrMissingRequiredField,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.modify(base)
			err := validatePlaceOrderRequest(req, "key-1")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("want %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestValidatePlaceOrderRequest_ExtraMetadataKeysAllowed(t *testing.T) {
	req := PlaceOrderRequest{
		UserID:    "user-001",
		ProductID: "prod-001",
		Quantity:  1,
		Amount:    2000,
		Metadata:  map[string]string{"address": "123 Main St", "floor": "3", "landmark": "near park"},
		Latitude:  "12.97",
		Longitude: "77.59",
	}
	if err := validatePlaceOrderRequest(req, "key-extra"); err != nil {
		t.Errorf("extra metadata keys should be allowed, got error: %v", err)
	}
}
