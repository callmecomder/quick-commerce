package repository

import (
	"context"

	"gorm.io/gorm"
)

type ctxKey struct{}

type TxManager struct {
	DB *gorm.DB
}

func NewTxManager(db *gorm.DB) *TxManager {
	return &TxManager{DB: db}
}

func (t *TxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return t.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, ctxKey{}, tx)
		return fn(txCtx)
	})
}

func GetDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(ctxKey{}).(*gorm.DB); ok {
		return tx
	}
	return nil
}

func (t *TxManager) InjectDB(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKey{}, t.DB)
}
