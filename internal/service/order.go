package service

import (
	"context"
	"encoding/json"
	"fmt"

	"quickcommerce/internal/domain"
	"quickcommerce/internal/payment"
	"quickcommerce/internal/repository"
)

// PlaceOrderRequest holds the validated input for placing an order.
type PlaceOrderRequest struct {
	UserID    string            `json:"user_id"`
	ProductID string            `json:"product_id"`
	Quantity  int               `json:"quantity"`
	Amount    int64             `json:"amount"`
	Metadata  map[string]string `json:"metadata"`
	Latitude  string            `json:"latitude"`
	Longitude string            `json:"longitude"`
}

// PlaceOrderResult contains the created order along with product display info.
type PlaceOrderResult struct {
	Order              *domain.Order
	ProductBrand       string
	ProductDescription string
}

// OrderService handles order placement with idempotency and stock consistency.
type OrderService struct {
	productRepo *repository.ProductRepo
	userRepo    *repository.UserRepo
	orderRepo   *repository.OrderRepo
	txManager   *repository.TxManager
	payment     payment.Payment
}

// NewOrderService creates an OrderService with the given dependencies.
func NewOrderService(
	pr *repository.ProductRepo,
	ur *repository.UserRepo,
	or *repository.OrderRepo,
	tx *repository.TxManager,
	pay payment.Payment,
) *OrderService {
	return &OrderService{
		productRepo: pr,
		userRepo:    ur,
		orderRepo:   or,
		txManager:   tx,
		payment:     pay,
	}
}

type txResult struct {
	order        *domain.Order
	productBrand string
	productDesc  string
	payFailed    bool
	failedOrder  *domain.Order
}

// PlaceOrder validates, checks idempotency, and places an order inside a DB transaction.
func (s *OrderService) PlaceOrder(ctx context.Context, req PlaceOrderRequest, idempotencyKey string) (*PlaceOrderResult, error) {
	if err := validatePlaceOrderRequest(req, idempotencyKey); err != nil {
		return nil, err
	}

	ctxWithDB := s.txManager.InjectDB(ctx)
	existing, err := s.orderRepo.GetByRequestID(ctxWithDB, idempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("idempotency check: %w", err)
	}
	if existing != nil {
		return s.buildReplayResult(ctxWithDB, existing)
	}

	res, txErr := s.executePlaceOrderTx(ctx, req, idempotencyKey)
	if txErr != nil {
		if res != nil && res.payFailed && res.failedOrder != nil {
			return s.persistFailedOrder(ctx, res)
		}
		return nil, txErr
	}

	return &PlaceOrderResult{
		Order:              res.order,
		ProductBrand:       res.productBrand,
		ProductDescription: res.productDesc,
	}, nil
}

func (s *OrderService) buildReplayResult(ctx context.Context, existing *domain.Order) (*PlaceOrderResult, error) {
	product, err := s.productRepo.GetByID(ctx, existing.ProductID)
	if err != nil {
		return nil, fmt.Errorf("fetch product for replay: %w", err)
	}
	return &PlaceOrderResult{
		Order:              existing,
		ProductBrand:       product.Brand,
		ProductDescription: product.Description,
	}, nil
}

func (s *OrderService) executePlaceOrderTx(ctx context.Context, req PlaceOrderRequest, requestID string) (*txResult, error) {
	res := &txResult{}

	txErr := s.txManager.WithinTx(ctx, func(txCtx context.Context) error {
		product, err := s.productRepo.GetForUpdate(txCtx, req.ProductID)
		if err != nil {
			return err
		}
		res.productBrand = product.Brand
		res.productDesc = product.Description

		user, err := s.userRepo.GetByID(txCtx, req.UserID)
		if err != nil {
			return err
		}
		if user.Status != domain.UserStatusActive {
			return domain.ErrUserInactive
		}

		expectedAmount := product.Amount * int64(req.Quantity)
		if req.Amount != expectedAmount {
			return domain.ErrAmountMismatch
		}

		if product.Quantity < req.Quantity {
			return domain.ErrOutOfStock
		}

		meta, err := buildOrderMetadata(req)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}

		order := &domain.Order{
			UserID:    req.UserID,
			ProductID: req.ProductID,
			Amount:    expectedAmount,
			Quantity:  req.Quantity,
			Metadata:  meta,
			RequestID: requestID,
		}

		_, payErr := s.payment.Charge(txCtx, expectedAmount, req.UserID, requestID)
		if payErr != nil {
			reason := payErr.Error()
			order.Status = domain.OrderStatusFailed
			order.FailureReason = &reason
			res.failedOrder = order
			res.payFailed = true
			return fmt.Errorf("payment: %w", payErr)
		}

		order.Status = domain.OrderStatusSuccess

		product.Quantity -= req.Quantity
		if err := s.productRepo.Update(txCtx, product); err != nil {
			return fmt.Errorf("update stock: %w", err)
		}

		if err := s.orderRepo.Create(txCtx, order); err != nil {
			return fmt.Errorf("create order: %w", err)
		}

		res.order = order
		return nil
	})

	return res, txErr
}

func (s *OrderService) persistFailedOrder(ctx context.Context, res *txResult) (*PlaceOrderResult, error) {
	dbCtx := s.txManager.InjectDB(ctx)
	if err := s.orderRepo.Create(dbCtx, res.failedOrder); err != nil {
		return nil, fmt.Errorf("create failed order: %w", err)
	}
	return &PlaceOrderResult{
		Order:              res.failedOrder,
		ProductBrand:       res.productBrand,
		ProductDescription: res.productDesc,
	}, nil
}

func buildOrderMetadata(req PlaceOrderRequest) ([]byte, error) {
	metaMap := make(map[string]interface{}, len(req.Metadata)+3)
	for k, v := range req.Metadata {
		metaMap[k] = v
	}
	metaMap["quantity"] = req.Quantity
	metaMap["latitude"] = req.Latitude
	metaMap["longitude"] = req.Longitude
	return json.Marshal(metaMap)
}
