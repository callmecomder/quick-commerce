# Quick-Commerce Implementation Plan (compact)

**Goal:** Go service: product search/get + order placement with idempotency & stock consistency.
**Stack:** Go 1.24, chi, GORM + MySQL 8, dockertest, mock payment. Layered (handler→service→repo).
Timestamps epoch-ms. Money paise (int64). Stock consistency = MySQL `FOR UPDATE` row lock in TX.

## File map
```
go.mod                                   module quickcommerce
cmd/server/main.go                       wire + migrate + seed + graceful shutdown
internal/config/config.go                env config + DSN
internal/domain/models.go                User, Product, Order, IdempotencyKey + enums
internal/domain/errors.go                sentinel errors
internal/payment/payment.go              Payment interface + MockPayment + ErrPaymentFailed
internal/repository/tx.go                TxManager (ctx-scoped *gorm.DB)
internal/repository/repos.go             gorm repos: product/user/order/idempotency
internal/service/product.go              ProductService
internal/service/order.go                OrderService (core flow)
internal/service/order_test.go           unit: happy/oos/replay/payfail/amount/user
internal/service/product_test.go         unit: search/get
internal/handler/dto.go                  request/response structs
internal/handler/product.go              product handlers
internal/handler/order.go                order handler
internal/handler/router.go               chi router
internal/httperr/httperr.go              error->HTTP map
internal/platform/id.go                  uuid id gen
test/e2e/e2e_test.go                     dockertest: concurrency no-oversell + idempotency
Dockerfile                               multistage
docker-compose.yml                       app + mysql
api/quick-commerce.http                  REST-client flow
api/quick-commerce.postman_collection.json
README.md
```

## Tasks
1. `go mod init quickcommerce` + add deps (chi, gorm, gorm mysql driver, google/uuid, dockertest, datatypes).
2. domain models + errors (GORM tags, `autoCreateTime:milli` / `autoUpdateTime:milli`).
3. payment interface + mock (success default; `ErrPaymentFailed` sentinel; FailMode for tests).
4. repository: TxManager (ctx tx injection) + gorm repos. `GetForUpdate` uses `clause.Locking{Strength:"UPDATE"}`.
5. ProductService + unit test (mocked repo).
6. OrderService core + unit tests (mocked repos, TxManager.WithinTx = call fn, mock payment). Flow per spec §5.
7. httperr map + DTOs + handlers + chi router.
8. main: config, gorm open, AutoMigrate, seed users+products if empty, graceful shutdown.
9. Dockerfile + docker-compose (mysql healthcheck-gated).
10. E2E dockertest: spin mysql, httptest server, 20 concurrent orders on stock=5 → exactly 5 success/0 left; idem replay returns same order.
11. api flow collection (.http + postman) ordered search→get→order→replay.
12. README (design, run, curl, deviations).

## Order flow (service.PlaceOrder)
- empty idem key → ErrMissingIdempotencyKey.
- idem.Get hit → return stored order (load via order.GetByID).
- WithinTx: GetForUpdate(product)→404; user active check; amount = product.amount*qty validate; stock<qty→ErrOutOfStock; payment.Charge; on payErr return it (rollback, no stock change); else decrement, create success order, create idem key.
- after tx: payment error → create `failed` order (no stock change), return it (HTTP 200). validation errors → 4xx.
