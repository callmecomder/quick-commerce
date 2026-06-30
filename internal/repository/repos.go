package repository

import (
	"context"
	"errors"

	"quickcommerce/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProductRepo struct{}

func NewProductRepo() *ProductRepo { return &ProductRepo{} }

func (r *ProductRepo) Search(ctx context.Context, name string) ([]domain.Product, error) {
	db := GetDB(ctx)
	var products []domain.Product
	q := db.WithContext(ctx)
	if name != "" {
		q = q.Where("description LIKE ? OR brand LIKE ?", "%"+name+"%", "%"+name+"%")
	}
	if err := q.Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

func (r *ProductRepo) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	db := GetDB(ctx)
	var p domain.Product
	if err := db.WithContext(ctx).Where("id = ?", id).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProductNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepo) GetForUpdate(ctx context.Context, id string) (*domain.Product, error) {
	db := GetDB(ctx)
	var p domain.Product
	if err := db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProductNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepo) Update(ctx context.Context, p *domain.Product) error {
	db := GetDB(ctx)
	return db.WithContext(ctx).Save(p).Error
}

type UserRepo struct{}

func NewUserRepo() *UserRepo { return &UserRepo{} }

func (r *UserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	db := GetDB(ctx)
	var u domain.User
	if err := db.WithContext(ctx).Where("id = ?", id).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

type OrderRepo struct{}

func NewOrderRepo() *OrderRepo { return &OrderRepo{} }

func (r *OrderRepo) Create(ctx context.Context, o *domain.Order) error {
	db := GetDB(ctx)
	return db.WithContext(ctx).Create(o).Error
}

func (r *OrderRepo) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	db := GetDB(ctx)
	var o domain.Order
	if err := db.WithContext(ctx).Where("id = ?", id).First(&o).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("order not found")
		}
		return nil, err
	}
	return &o, nil
}

func (r *OrderRepo) GetByRequestID(ctx context.Context, requestID string) (*domain.Order, error) {
	db := GetDB(ctx)
	var o domain.Order
	if err := db.WithContext(ctx).Where("request_id = ?", requestID).First(&o).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}
