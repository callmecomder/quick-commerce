package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"quickcommerce/internal/config"
	"quickcommerce/internal/domain"
	"quickcommerce/internal/handler"
	"quickcommerce/internal/payment"
	"quickcommerce/internal/repository"
	"quickcommerce/internal/service"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	cfg := config.Load()

	db, err := gorm.Open(mysql.Open(cfg.DBDsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}

	if err := db.AutoMigrate(
		&domain.User{},
		&domain.Product{},
		&domain.Order{},
	); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	seed(db)

	txManager := repository.NewTxManager(db)
	productRepo := repository.NewProductRepo()
	userRepo := repository.NewUserRepo()
	orderRepo := repository.NewOrderRepo()
	mockPayment := payment.NewMockPayment()

	productSvc := service.NewProductService(productRepo, txManager)
	orderSvc := service.NewOrderService(productRepo, userRepo, orderRepo, txManager, mockPayment)

	productHandler := handler.NewProductHandler(productSvc)
	orderHandler := handler.NewOrderHandler(orderSvc)

	router := handler.NewRouter(productHandler, orderHandler)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		log.Printf("server starting on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
	log.Println("server stopped")
}

func seed(db *gorm.DB) {
	var count int64
	db.Model(&domain.User{}).Count(&count)
	if count > 0 {
		return
	}

	users := []domain.User{
		{ID: "user-001", Contact: "9876543210", Email: "alice@example.com", Status: domain.UserStatusActive},
		{ID: "user-002", Contact: "9876543211", Email: "bob@example.com", Status: domain.UserStatusActive},
		{ID: "user-003", Contact: "9876543212", Email: "charlie@example.com", Status: domain.UserStatusInactive},
		{ID: "user-004", Contact: "9876543213", Email: "diana@example.com", Status: domain.UserStatusActive},
		{ID: "user-005", Contact: "9876543214", Email: "eve@example.com", Status: domain.UserStatusCreated},
	}
	for _, u := range users {
		db.Create(&u)
	}

	products := []domain.Product{
		{ID: "prod-001", Description: "45gms", Brand: "Lays Classic Salted", Amount: 2000, Quantity: 100, Metadata: mustJSON(map[string]interface{}{"prev_amount": 1500})},
		{ID: "prod-002", Description: "500ml", Brand: "Coca Cola", Amount: 4000, Quantity: 50, Metadata: mustJSON(map[string]interface{}{"prev_amount": 3500})},
		{ID: "prod-003", Description: "1L", Brand: "Amul Toned Milk", Amount: 6800, Quantity: 200, Metadata: mustJSON(map[string]interface{}{"prev_amount": 6000})},
		{ID: "prod-004", Description: "200gms", Brand: "Uncle Chips Spicy", Amount: 3000, Quantity: 80, Metadata: mustJSON(map[string]interface{}{"prev_amount": 2500})},
		{ID: "prod-005", Description: "100gms pack of 4", Brand: "Maggi 2-Minute Noodles", Amount: 1400, Quantity: 150, Metadata: mustJSON(map[string]interface{}{"prev_amount": 1200})},
		{ID: "prod-006", Description: "750ml", Brand: "Pepsi", Amount: 3800, Quantity: 60, Metadata: mustJSON(map[string]interface{}{"prev_amount": 3200})},
		{ID: "prod-007", Description: "400gms", Brand: "Britannia Good Day", Amount: 5500, Quantity: 90, Metadata: mustJSON(map[string]interface{}{"prev_amount": 5000})},
		{ID: "prod-008", Description: "1kg", Brand: "Aashirvaad Atta", Amount: 18000, Quantity: 120, Metadata: mustJSON(map[string]interface{}{"prev_amount": 16000})},
		{ID: "prod-009", Description: "200ml", Brand: "Paper Boat Aam Panna", Amount: 3000, Quantity: 70, Metadata: mustJSON(map[string]interface{}{"prev_amount": 2500})},
		{ID: "prod-010", Description: "500gms", Brand: "Haldiram Aloo Bhujia", Amount: 9500, Quantity: 40, Metadata: mustJSON(map[string]interface{}{"prev_amount": 8500})},
	}
	for _, p := range products {
		db.Create(&p)
	}

	fmt.Println("seeded users and products")
}

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
