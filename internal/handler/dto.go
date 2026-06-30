package handler

type SearchProductsResponse struct {
	Products []ProductDTO `json:"products"`
}

type ProductDTO struct {
	ProductID   string `json:"product_id"`
	Description string `json:"description"`
	Brand       string `json:"brand"`
	Quantity    int    `json:"quantity"`
	Amount      int64  `json:"amount"`
}

type PlaceOrderRequest struct {
	UserID    string            `json:"user_id"`
	ProductID string            `json:"product_id"`
	Quantity  int               `json:"quantity"`
	Amount    int64             `json:"amount"`
	Metadata  map[string]string `json:"metadata"`
	Latitude  string            `json:"latitude"`
	Longitude string            `json:"longitude"`
}

type OrderResponse struct {
	ID                 string  `json:"id"`
	UserID             string  `json:"user_id,omitempty"`
	Status             string  `json:"status"`
	ProductID          string  `json:"product_id"`
	ProductName        string  `json:"product_name"`
	ProductDescription string  `json:"product_description"`
	Quantity           int     `json:"quantity"`
	Amount             int64   `json:"amount"`
	FailureReason      *string `json:"failure_reason,omitempty"`
	CreatedAt          int64   `json:"created_at"`
}
