package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"quickcommerce/internal/domain"
	"quickcommerce/internal/handler"
	"quickcommerce/internal/payment"
	"quickcommerce/internal/repository"
	"quickcommerce/internal/service"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	testDB      *gorm.DB
	testServer  *httptest.Server
	mockPayment *payment.MockPayment
)

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("dockertest pool: %v", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "mysql",
		Tag:        "8.0",
		Env: []string{
			"MYSQL_ROOT_PASSWORD=testpass",
			"MYSQL_DATABASE=testdb",
		},
	}, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
		hc.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("start mysql: %v", err)
	}
	resource.Expire(120)

	dsn := fmt.Sprintf("root:testpass@tcp(localhost:%s)/testdb?charset=utf8mb4&parseTime=True&loc=Local",
		resource.GetPort("3306/tcp"))

	pool.MaxWait = 90 * time.Second
	if err := pool.Retry(func() error {
		db, openErr := gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if openErr != nil {
			return openErr
		}
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		if err := sqlDB.Ping(); err != nil {
			sqlDB.Close()
			return err
		}
		testDB = db
		return nil
	}); err != nil {
		log.Fatalf("wait mysql: %v", err)
	}

	if err := testDB.AutoMigrate(&domain.User{}, &domain.Product{}, &domain.Order{}); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	seedTestData(testDB)

	mockPayment = payment.NewMockPayment()
	txManager := repository.NewTxManager(testDB)
	productRepo := repository.NewProductRepo()
	userRepo := repository.NewUserRepo()
	orderRepo := repository.NewOrderRepo()

	productSvc := service.NewProductService(productRepo, txManager)
	orderSvc := service.NewOrderService(productRepo, userRepo, orderRepo, txManager, mockPayment)

	productHandler := handler.NewProductHandler(productSvc)
	orderHandler := handler.NewOrderHandler(orderSvc)

	router := handler.NewRouter(productHandler, orderHandler)
	testServer = httptest.NewServer(router)

	code := m.Run()

	testServer.Close()
	pool.Purge(resource)
	os.Exit(code)
}

func seedTestData(db *gorm.DB) {
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
		{ID: "prod-001", Description: "45gms", Brand: "Lays Classic Salted", Amount: 2000, Quantity: 100},
		{ID: "prod-002", Description: "500ml", Brand: "Coca Cola", Amount: 4000, Quantity: 50},
		{ID: "prod-003", Description: "1L", Brand: "Amul Toned Milk", Amount: 6800, Quantity: 200},
	}
	for _, p := range products {
		db.Create(&p)
	}
}

func resetStock(productID string, qty int) {
	testDB.Model(&domain.Product{}).Where("id = ?", productID).Update("quantity", qty)
}

type orderReq struct {
	UserID    string            `json:"user_id"`
	ProductID string            `json:"product_id"`
	Quantity  int               `json:"quantity"`
	Amount    int64             `json:"amount"`
	Metadata  map[string]string `json:"metadata"`
	Latitude  string            `json:"latitude"`
	Longitude string            `json:"longitude"`
}

type orderResp struct {
	ID                 string  `json:"id"`
	UserID             string  `json:"user_id"`
	Status             string  `json:"status"`
	ProductID          string  `json:"product_id"`
	ProductName        string  `json:"product_name"`
	ProductDescription string  `json:"product_description"`
	Quantity           int     `json:"quantity"`
	Amount             int64   `json:"amount"`
	FailureReason      *string `json:"failure_reason"`
	CreatedAt          int64   `json:"created_at"`
}

func placeOrder(t *testing.T, req orderReq, idemKey string) (*http.Response, orderResp) {
	t.Helper()
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest(http.MethodPost, testServer.URL+"/v1/orders", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	if idemKey != "" {
		httpReq.Header.Set("Idempotency-Key", idemKey)
	}
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		t.Fatalf("http request: %v", err)
	}
	var result orderResp
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()
	return resp, result
}

func defaultReq() orderReq {
	return orderReq{
		UserID:    "user-001",
		ProductID: "prod-001",
		Quantity:  1,
		Amount:    2000,
		Metadata:  map[string]string{"address": "123 Main St, Bangalore"},
		Latitude:  "12.9716",
		Longitude: "77.5946",
	}
}

func uid(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func setPaymentFail(t *testing.T, fail bool) {
	t.Helper()
	mockPayment.ShouldFail = fail
	t.Cleanup(func() { mockPayment.ShouldFail = false })
}

// ---- Product tests ----

func TestSearchProducts(t *testing.T) {
	resp, err := http.Get(testServer.URL + "/v1/products?product_name=Lays")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200 got %d", resp.StatusCode)
	}

	var result struct {
		Products []struct {
			ProductID string `json:"product_id"`
			Brand     string `json:"brand"`
			Amount    int64  `json:"amount"`
			Quantity  int    `json:"quantity"`
		} `json:"products"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Products) == 0 {
		t.Fatal("no products returned")
	}
	if result.Products[0].ProductID != "prod-001" {
		t.Errorf("want prod-001 got %s", result.Products[0].ProductID)
	}
	if result.Products[0].Brand != "Lays Classic Salted" {
		t.Errorf("unexpected brand: %s", result.Products[0].Brand)
	}
}

func TestGetProductByID(t *testing.T) {
	resp, err := http.Get(testServer.URL + "/v1/products/prod-002")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200 got %d", resp.StatusCode)
	}

	var result struct {
		ProductID string `json:"product_id"`
		Brand     string `json:"brand"`
		Amount    int64  `json:"amount"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.ProductID != "prod-002" {
		t.Errorf("want prod-002 got %s", result.ProductID)
	}
	if result.Amount != 4000 {
		t.Errorf("want 4000 got %d", result.Amount)
	}
}

func TestGetProductByID_NotFound(t *testing.T) {
	resp, err := http.Get(testServer.URL + "/v1/products/nonexistent-prod")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 got %d", resp.StatusCode)
	}
}

// ---- Order success + response shape ----

func TestPlaceOrder_Success(t *testing.T) {
	resp, result := placeOrder(t, defaultReq(), uid("success"))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200 got %d", resp.StatusCode)
	}
	if result.Status != "success" {
		t.Errorf("want success got %s", result.Status)
	}
	if len(result.ID) != 14 {
		t.Errorf("order ID must be 14 chars, got %d (%s)", len(result.ID), result.ID)
	}
	if result.ProductName != "Lays Classic Salted" {
		t.Errorf("want product_name 'Lays Classic Salted' got %s", result.ProductName)
	}
	if result.ProductDescription != "45gms" {
		t.Errorf("want product_description '45gms' got %s", result.ProductDescription)
	}
	if result.Amount != 2000 {
		t.Errorf("want amount 2000 got %d", result.Amount)
	}
	if result.UserID != "user-001" {
		t.Errorf("want user-001 got %s", result.UserID)
	}
	if result.CreatedAt == 0 {
		t.Error("created_at must be set")
	}
	if result.FailureReason != nil {
		t.Errorf("failure_reason must be nil on success, got %s", *result.FailureReason)
	}
}

func TestPlaceOrder_MultipleQuantity(t *testing.T) {
	req := defaultReq()
	req.Quantity = 3
	req.Amount = 6000

	resp, result := placeOrder(t, req, uid("multi"))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200 got %d", resp.StatusCode)
	}
	if result.Quantity != 3 {
		t.Errorf("want 3 got %d", result.Quantity)
	}
	if result.Amount != 6000 {
		t.Errorf("want 6000 got %d", result.Amount)
	}
}

// ---- Validation errors ----

func TestPlaceOrder_MissingIdempotencyKey(t *testing.T) {
	resp, _ := placeOrder(t, defaultReq(), "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 got %d", resp.StatusCode)
	}
}

func TestPlaceOrder_InvalidQuantity(t *testing.T) {
	req := defaultReq()
	req.Quantity = 0
	req.Amount = 0
	resp, _ := placeOrder(t, req, uid("qty"))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 got %d", resp.StatusCode)
	}
}

func TestPlaceOrder_AmountMismatch(t *testing.T) {
	req := defaultReq()
	req.Amount = 9999
	resp, _ := placeOrder(t, req, uid("amount"))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 got %d", resp.StatusCode)
	}
}

func TestPlaceOrder_MissingAddress(t *testing.T) {
	req := defaultReq()
	req.Metadata = map[string]string{"note": "no address here"}
	resp, _ := placeOrder(t, req, uid("noaddr"))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 got %d", resp.StatusCode)
	}
}

func TestPlaceOrder_MissingLatLong(t *testing.T) {
	req := defaultReq()
	req.Latitude = ""
	resp, _ := placeOrder(t, req, uid("noll"))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 got %d", resp.StatusCode)
	}
}

// ---- Domain errors ----

func TestPlaceOrder_InactiveUser(t *testing.T) {
	req := defaultReq()
	req.UserID = "user-003"
	resp, _ := placeOrder(t, req, uid("inactive"))
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("want 409 got %d", resp.StatusCode)
	}
}

func TestPlaceOrder_UserNotCreatedStatus(t *testing.T) {
	req := defaultReq()
	req.UserID = "user-005"
	resp, _ := placeOrder(t, req, uid("created-status"))
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("want 409 got %d", resp.StatusCode)
	}
}

func TestPlaceOrder_UserNotFound(t *testing.T) {
	req := defaultReq()
	req.UserID = "user-999"
	resp, _ := placeOrder(t, req, uid("usernf"))
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 got %d", resp.StatusCode)
	}
}

func TestPlaceOrder_ProductNotFound(t *testing.T) {
	req := defaultReq()
	req.ProductID = "prod-999"
	resp, _ := placeOrder(t, req, uid("prodnf"))
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 got %d", resp.StatusCode)
	}
}

func TestPlaceOrder_OutOfStock(t *testing.T) {
	resetStock("prod-001", 0)
	defer resetStock("prod-001", 100)

	resp, _ := placeOrder(t, defaultReq(), uid("oos"))
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("want 409 got %d", resp.StatusCode)
	}
}

// ---- Payment failure ----

func TestPlaceOrder_PaymentFailure(t *testing.T) {
	setPaymentFail(t, true)

	key := uid("payfail")

	var prodBefore domain.Product
	testDB.First(&prodBefore, "id = ?", "prod-001")
	stockBefore := prodBefore.Quantity

	resp, result := placeOrder(t, defaultReq(), key)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200 got %d", resp.StatusCode)
	}
	if result.Status != "failed" {
		t.Errorf("want failed got %s", result.Status)
	}
	if result.FailureReason == nil || *result.FailureReason == "" {
		t.Error("failure_reason must be set")
	}

	var prodAfter domain.Product
	testDB.First(&prodAfter, "id = ?", "prod-001")
	if prodAfter.Quantity != stockBefore {
		t.Errorf("stock changed: before=%d after=%d", stockBefore, prodAfter.Quantity)
	}

	var order domain.Order
	testDB.Where("request_id = ?", key).First(&order)
	if order.ID == "" {
		t.Error("failed order not persisted")
	}
	if order.Status != domain.OrderStatusFailed {
		t.Errorf("order status must be failed, got %s", order.Status)
	}
}

// ---- Idempotency ----

func TestIdempotency_ReplayReturnsSameOrder(t *testing.T) {
	key := uid("replay")

	resp1, result1 := placeOrder(t, defaultReq(), key)
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("first: want 200 got %d", resp1.StatusCode)
	}
	if result1.Status != "success" {
		t.Fatalf("first: want success got %s", result1.Status)
	}

	resp2, result2 := placeOrder(t, defaultReq(), key)
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("replay: want 200 got %d", resp2.StatusCode)
	}

	if result1.ID != result2.ID {
		t.Errorf("replay returned different order ID: %s vs %s", result1.ID, result2.ID)
	}
	if result1.Status != result2.Status {
		t.Errorf("replay status differs: %s vs %s", result1.Status, result2.Status)
	}
	if result2.ProductName == "" {
		t.Error("replay must include product_name")
	}
}

func TestIdempotency_FailedOrderReplay(t *testing.T) {
	key := uid("failreplay")

	setPaymentFail(t, true)
	resp1, result1 := placeOrder(t, defaultReq(), key)
	mockPayment.ShouldFail = false

	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("first: want 200 got %d", resp1.StatusCode)
	}
	if result1.Status != "failed" {
		t.Fatalf("first: want failed got %s", result1.Status)
	}

	resp2, result2 := placeOrder(t, defaultReq(), key)
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("replay: want 200 got %d", resp2.StatusCode)
	}
	if result1.ID != result2.ID {
		t.Errorf("failed replay returned different ID: %s vs %s", result1.ID, result2.ID)
	}
	if result2.Status != "failed" {
		t.Errorf("replay of failed order must be failed, got %s", result2.Status)
	}
}

// ---- Concurrency ----

func TestConcurrentOrders_NoOversell(t *testing.T) {
	const stock = 5
	const goroutines = 20

	resetStock("prod-001", stock)
	defer resetStock("prod-001", 100)

	var (
		mu         sync.Mutex
		successes  int
		oos        int
		unexpected []string
		wg         sync.WaitGroup
	)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("oversell-%d-%d", time.Now().UnixNano(), n)
			resp, result := placeOrder(t, defaultReq(), key)
			mu.Lock()
			defer mu.Unlock()
			switch resp.StatusCode {
			case http.StatusOK:
				if result.Status == "success" {
					successes++
				}
			case http.StatusConflict:
				oos++
			default:
				unexpected = append(unexpected, fmt.Sprintf("g%d: status=%d", n, resp.StatusCode))
			}
		}(i)
	}

	wg.Wait()

	if len(unexpected) > 0 {
		t.Errorf("unexpected responses: %v", unexpected)
	}
	if successes != stock {
		t.Errorf("want exactly %d successes (= stock), got %d", stock, successes)
	}
	if oos != goroutines-stock {
		t.Errorf("want %d out-of-stock, got %d", goroutines-stock, oos)
	}

	var prod domain.Product
	testDB.First(&prod, "id = ?", "prod-001")
	if prod.Quantity != 0 {
		t.Errorf("final stock must be 0, got %d", prod.Quantity)
	}
}

func TestConcurrent_SameIdempotencyKey(t *testing.T) {
	const goroutines = 10
	key := uid("samekey")

	var (
		mu  sync.Mutex
		ids []string
		wg  sync.WaitGroup
	)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, result := placeOrder(t, defaultReq(), key)
			if resp.StatusCode == http.StatusOK {
				mu.Lock()
				ids = append(ids, result.ID)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if len(ids) == 0 {
		t.Fatal("no successful responses")
	}

	first := ids[0]
	for _, id := range ids[1:] {
		if id != first {
			t.Errorf("concurrent same key produced different order IDs: %s vs %s", first, id)
		}
	}
}
