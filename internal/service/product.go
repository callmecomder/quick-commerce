package service

import (
	"context"

	"quickcommerce/internal/domain"
	"quickcommerce/internal/repository"
)

type ProductService struct {
	productRepo *repository.ProductRepo
	txManager   *repository.TxManager
}

func NewProductService(pr *repository.ProductRepo, tx *repository.TxManager) *ProductService {
	return &ProductService{productRepo: pr, txManager: tx}
}

func (s *ProductService) Search(ctx context.Context, name string) ([]domain.Product, error) {
	ctx = s.txManager.InjectDB(ctx)
	return s.productRepo.Search(ctx, name)
}

func (s *ProductService) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	ctx = s.txManager.InjectDB(ctx)
	return s.productRepo.GetByID(ctx, id)
}
