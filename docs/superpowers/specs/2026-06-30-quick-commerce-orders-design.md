# Quick-Commerce: Orders & Product Search — Design

**Date:** 2026-06-30
**Status:** Approved
**Scope:** `POST /v1/orders` (place order, direct checkout), `GET /v1/products` (search),
`GET /v1/products/{product_id}` (get one). Payments flow is OUT of scope (mock interface).

## 1. Goals & Non-Goals

### Functional
- Search products by name (lat/long accepted in request, single-store assumption for v1).
- Get product by id.
- Place an order ("Buy Now" = direct checkout) with idempotency.

### Non-Functional (from spec)
- **Consistency >> Availability.** No overselling of stock under concurrency.
- Fault tolerance: payment failure/timeout → order recorded as `failed` with `failure_reason`, stock not consumed.
- Scaling: stateless app, horizontal scale; consistency enforced at the DB row level (not in app memory).

### Out of Scope
- Real payments (mocked, always success).
- Cart, multi-item orders, user creation API, product creation API (users/products are seeded).
- Geo ranking / inventory-by-location.

## 2. Architecture

Layered (Handler → Service → Repository), matching the spec's Server → Core → Repo split.

```
cmd/server/                main(): config, DB connect, migrate, wire deps, start HTTP
internal/config/           env-based config
internal/domain/           entities + status enums + domain errors
internal/repository/       GORM data access (products, orders, users, idempotency)
internal/service/          business logic: ProductService, OrderService (the order core)
internal/payment/          Payment interface + mock impl (always success)
internal/handler/          HTTP handlers (chi router), request/response DTOs, error mapping
internal/httperr/          central error -> HTTP status mapping
migrations/                SQL DDL + seed data (001_create_tables.sql, 002_seed_data.sql)
```

**Stack:** Go 1.24, chi (router), GORM + MySQL driver, MySQL 8 (Docker). Tests: stdlib `testing` +
`dockertest` (spins MySQL container inside `go test`).

## 3. Data Model (MySQL via GORM)

### users
| col | type | notes |
|---|---|---|
| id | varchar PK | |
| contact | varchar | |
| email | varchar | |
| status | enum(created/active/inactive) | order allowed only if `active` |
| created_at, updated_at | bigint (epoch ms) | |

### products
| col | type | notes |
|---|---|---|
| id | varchar PK (indexed) | |
| description | varchar | |
| amount | bigint | price in paise (integer money, no float) |
| quantity | int | available stock |
| metadata | json | prev amount etc. |
| created_at, updated_at | bigint (epoch ms) | |

### orders
| col | type | notes |
|---|---|---|
| id | varchar(14) PK | 14-char hex string (crypto/rand) |
| user_id | varchar (indexed) | |
| product_id | varchar | |
| amount | bigint | computed = product.amount * qty |
| quantity | int | |
| status | enum(success/failed) | |
| metadata | json | request `metadata` map keys + {quantity, latitude, longitude} |
| failure_reason | varchar nullable | set when status=failed |
| request_id | varchar(255), **UNIQUE** | `"pay_" + Idempotency-Key` — payment correlation AND idempotency token |
| created_at | bigint (epoch ms) | |

### Idempotency (no separate table)

The `Idempotency-Key` header is **not** stored in a separate `idempotency_keys` table. Instead:
- `orders.request_id = "pay_" + <Idempotency-Key>` with a UNIQUE index.
- Replay: `SELECT * FROM orders WHERE request_id = "pay_" + key` → return existing order, skip payment.
- Race: UNIQUE index makes a concurrent duplicate INSERT fail; the loser resolves to the winner's order via the same SELECT.

Rationale: single source of truth, one fewer table to migrate, replay is a single indexed lookup.

Money stored as integer paise everywhere. No floats.

**Timestamps:** all `created_at` / `updated_at` columns are `bigint` = **Unix epoch milliseconds** (int64),
not SQL datetime. GORM models use `int64` fields, set in BeforeCreate/BeforeUpdate hooks.

### Migrations

SQL files in `migrations/`:
- `001_create_tables.sql` — DDL for all 4 tables with proper types, indexes, comments.
- `002_seed_data.sql` — 5 users (3 active, 1 inactive, 1 created) + 10 products with realistic Indian FMCG items.

Run manually: `mysql -u root -p < migrations/001_create_tables.sql && mysql -u root -p < migrations/002_seed_data.sql`
Or use `docker compose up` which auto-migrates via GORM + seeds on empty DB.

### Seed Data

| ID | Brand | Amount (paise) | Stock |
|---|---|---|---|
| prod-001 | Lays Classic Salted | 2000 (₹20) | 100 |
| prod-002 | Coca Cola | 4000 (₹40) | 50 |
| prod-003 | Amul Toned Milk | 6800 (₹68) | 200 |
| prod-004 | Uncle Chips Spicy | 3000 (₹30) | 80 |
| prod-005 | Maggi 2-Minute Noodles | 1400 (₹14) | 150 |
| prod-006 | Pepsi | 3800 (₹38) | 60 |
| prod-007 | Britannia Good Day | 5500 (₹55) | 90 |
| prod-008 | Aashirvaad Atta | 18000 (₹180) | 120 |
| prod-009 | Paper Boat Aam Panna | 3000 (₹30) | 70 |
| prod-010 | Haldiram Aloo Bhujia | 9500 (₹95) | 40 |

Users: `user-001` Alice (active), `user-002` Bob (active), `user-003` Charlie (inactive),
`user-004` Diana (active), `user-005` Eve (created).

## 4. APIs

### GET /v1/products
Query params: `product_name` (substring match), `latitude`, `longitude` (accepted, unused in v1 ranking).
Response: `{ "products": [ { product_id, quantity, amount, description, brand } ] }`.

### GET /v1/products/{product_id}
404 if not found. Returns single product.

### POST /v1/orders
Header: `Idempotency-Key` (required, unique per logical order attempt).

Request body — **all fields mandatory** (else 400):
```json
{
  "user_id":   "user-001",
  "product_id":"prod-001",
  "quantity":  1,
  "amount":    2000,
  "metadata":  { "address": "123 Main St, Bangalore" },
  "latitude":  "12.9716",
  "longitude": "77.5946"
}
```

Validation:
- `user_id`, `product_id` non-empty
- `quantity > 0`, `amount > 0`
- `latitude`, `longitude` non-empty
- `metadata` is a `map[string]string`, non-empty, and must contain a non-empty `address` key (extra keys allowed, persisted as-is)
- `amount` is recomputed server-side and must equal `product.amount × quantity` (else 400)

Response:
```json
{
  "id":                  "a1b2c3d4e5f607",
  "user_id":             "user-001",
  "status":              "success",
  "product_id":          "prod-001",
  "product_name":        "Lays Classic Salted",
  "product_description": "45gms",
  "quantity":            1,
  "amount":              2000,
  "created_at":          1782700000000
}
```

- `id` is a **14-char hex string** (crypto/rand), not a UUID.
- `product_name` (brand) and `product_description` are returned for client display convenience — no extra round-trip to `GET /v1/products/{id}` needed.
- On payment failure: HTTP 200, `"status": "failed"`, `"failure_reason": "<reason>"`, stock untouched.

## 5. Order Flow (consistency core)

```
Handler:
  - decode body, validate required fields
  - require Idempotency-Key header (else 400)
  - call OrderService.PlaceOrder(ctx, req, idemKey)

OrderService.PlaceOrder:
  0. Validate all fields mandatory (user_id, product_id, quantity>0, amount>0, latitude, longitude, metadata.address).
  1. requestID = "pay_" + idempotencyKey
  2. Idempotency precheck (read-only, outside TX):
        SELECT * FROM orders WHERE request_id = requestID
        - found? load product (for product_name / product_description in response), return order.
  3. db.Transaction(func(tx){
       4. SELECT * FROM products WHERE id=? FOR UPDATE   // GORM clause.Locking{Strength:"UPDATE"}
          - row lock = the "mutex keyed by product_id" from the spec
       5. product not found -> ErrProductNotFound (404), rollback
       6. validate user: exists AND status=active -> else ErrUserInvalid (404/409)
       7. recompute amount = product.amount * quantity; if client amount mismatches -> ErrAmountMismatch (400)
       8. if product.quantity < requested -> ErrOutOfStock (409), rollback
       9. build order { request_id: requestID, metadata: <client metadata map> + lat/long/qty }
      10. payment.Charge(amount, user_id, requestID)
            - on error/timeout: mark order.status=failed, failure_reason; remember it; RETURN error
              -> TX rolls back (stock untouched). Failed order is persisted OUTSIDE the rolled-back TX
                 (so the client gets the failure_reason but stock isn't consumed).
      11. on success: decrement stock, INSERT order (status=success) — UNIQUE(request_id) makes a
          concurrent same-key insert fail; loser falls back to the SELECT in step 2 on retry.
     }) // COMMIT on success, ROLLBACK on any returned error
  12. map result/error to HTTP
```

### State machine
Order status transitions guarded in service (no library):
`created (transient) → success` on payment success, `created → failed` on payment error/timeout.
Only these transitions allowed.

### Concurrency guarantee
Two concurrent orders for the same product serialize on the `FOR UPDATE` row lock inside the TX.
Second waits for first to commit, then re-reads decremented quantity. No oversell. This is enforced
by MySQL/InnoDB, so it holds across multiple app instances (horizontal scale safe).

### Idempotency guarantee
- `Idempotency-Key` header is stored as `orders.request_id = "pay_" + key` (UNIQUE indexed). No separate `idempotency_keys` table.
- First request with a given key inserts the order; replay returns the same order, never charges twice.
- Race (two requests, same key, simultaneously): the UNIQUE index on `orders.request_id` makes the second INSERT fail; the loser's next read resolves to the winner's order via `SELECT * FROM orders WHERE request_id = "pay_" + key`.

## 6. Payment Abstraction

```go
type Payment interface {
    Charge(ctx, amount int64, userID, requestID string) (PaymentResult, error)
}
```
Mock impl returns success always (configurable failure available behind a flag for tests of the
failed-order path, but default = success per decision).

## 7. Error Handling

Central map (httperr):
| domain error | HTTP |
|---|---|
| validation / amount mismatch / missing idem key | 400 |
| product not found / user not found | 404 |
| out of stock | 409 |
| payment failure (recorded failed order) | 200 with status=failed body OR 502 — return 200 + failed order so client sees failure_reason |
| internal | 500 |
Idempotent replay → 200 with stored order.

## 8. Testing

### Unit (service layer, mocked repo + mock payment)
- PlaceOrder happy path → success order, stock decremented.
- Out of stock → 409, no order success, stock unchanged.
- User inactive/missing → error, rollback.
- Amount mismatch → 400.
- Payment failure → order failed + failure_reason, stock NOT consumed.
- Idempotency replay → same order returned, payment called once.
- Product search + get-by-id.

### E2E integration (dockertest — real MySQL container)
- Boot MySQL via dockertest, migrate, seed one product with quantity=N.
- Fire M concurrent PlaceOrder calls (M > N) for that product.
- Assert: exactly N succeed, rest get out-of-stock; final stock == 0; no oversell.
- Assert: replaying a successful order's idempotency key returns same order_id, stock unchanged.
This is the "at least 1 end-to-end test covering the complete flow" deliverable.

## 9. Deliverables

- Source repo (this layout).
- **Dockerfile** — multi-stage (build static binary → slim runtime).
- **docker-compose.yml** — app + mysql, healthcheck-gated startup, app auto-migrates + seeds on boot.
- **README.md** — design summary, run instructions, curl examples, and any deviations from the
  original discussion.
- **API flow collection** — to manually check/demonstrate the API flow against a running server:
  - `api/quick-commerce.http` — VS Code/JetBrains REST-client file: requests ordered as the real
    flow (search products → get product by id → place order with `Idempotency-Key` → replay same key
    to show idempotency). Uses `@baseUrl` + response-captured variables (e.g. capture `product_id`
    from search, feed into order) so the sequence is click-through runnable.
  - `api/quick-commerce.postman_collection.json` — equivalent Postman collection with the same
    ordered requests + a collection variable for `idempotencyKey`.
- `go test ./...` runs unit + e2e (dockertest needs Docker daemon).

## 10. Deviations from original spec (to note in README)
- Money stored as integer paise (avoids float rounding) — spec showed plain amounts.
- "Mutex lock keyed by product_id" implemented as MySQL `SELECT ... FOR UPDATE` row lock rather than
  an in-process mutex, so consistency holds across horizontally-scaled instances.
- Server recomputes `amount` from product price; client `amount` is validated, not trusted.
- Payment failure returns HTTP 200 with a `failed` order (so client receives `failure_reason`) rather
  than a 5xx.
- **Order ID is a 14-char hex string** (crypto/rand), not a UUID — short, URL-safe.
- **No separate `idempotency_keys` table.** Idempotency-Key is stored as `orders.request_id = "pay_" + key` with a UNIQUE index. One table, one source of truth.
- **All order request fields are mandatory** — `user_id`, `product_id`, `quantity`, `amount`, `latitude`, `longitude`, `metadata.address`.
- **`metadata` is a `map[string]string`** (must contain non-empty `address`). Extra keys are persisted on the order's `metadata` JSON column.
- **Order response includes `product_name` and `product_description`** so clients render the order without a follow-up `GET /v1/products/{id}`.
