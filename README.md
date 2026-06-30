# Quick Commerce — Orders & Product Search

Go service for quick-commerce: product search, get-by-id, and order placement with stock consistency + idempotency.

## Design

**Architecture:** Layered — Handler → Service → Repository (matches Server → Core → Repo from spec).

**Stack:** Go 1.24, chi (router), GORM + MySQL 8, dockertest (E2E tests).

### Order Flow (consistency core)

```
POST /v1/orders (Idempotency-Key header required)
  → idempotency check: SELECT * FROM orders WHERE request_id = "pay_" + key
      → found? return stored order (no re-charge)
  → BEGIN TX
    → SELECT product FOR UPDATE (row lock = mutex on product_id)
    → validate user active
    → validate amount = product.price × quantity
    → check stock >= requested (else 409)
    → payment.Charge (mock, always success)
    → decrement stock, create order (status=success, request_id="pay_<key>")
  → COMMIT
```

- **No oversell:** MySQL `SELECT ... FOR UPDATE` row lock serializes concurrent orders for same product. Holds across horizontally-scaled app instances.
- **Idempotency:** `Idempotency-Key` header stored as `orders.request_id` with a UNIQUE index. Replay returns original order, no double-charge.
- **Payment failure:** order created as `failed` with `failure_reason`, stock NOT consumed (TX rollback, failed order persisted outside TX so client sees the reason).
- **Order ID:** 14-character hex string (e.g. `a1b2c3d4e5f607`), generated via crypto/rand — short and URL-safe.

### Test Case Document

https://docs.google.com/document/d/1G28VMi17I8pSVN8Zspumdaku0b38dFeo/edit

### Excalidraw Design Discussed

<img width="6385" height="8063" alt="Untitled-2026-06-30-0943 excalidraw (1)" src="https://github.com/user-attachments/assets/c18e988c-0a75-40ed-b6f0-df31e176e61c" />


### Tables

All timestamps are **Unix epoch milliseconds** (int64). Money stored as **integer paise** (no floats).

#### users
| Column | Type | Notes |
|---|---|---|
| id | varchar(36) PK | UUID |
| contact | varchar(20) | phone |
| email | varchar(255) | |
| status | varchar(20) | `created` / `active` / `inactive` — order allowed only if `active` |
| created_at | bigint | epoch ms |
| updated_at | bigint | epoch ms |

#### products
| Column | Type | Notes |
|---|---|---|
| id | varchar(36) PK | UUID |
| description | varchar(500) | e.g. "45gms", "500ml" |
| brand | varchar(100) | e.g. "Lays Classic Salted" |
| amount | bigint | price in paise (2000 = ₹20.00) |
| quantity | int | available stock |
| metadata | json | `{"prev_amount": 1500}` |
| created_at | bigint | epoch ms |
| updated_at | bigint | epoch ms |

#### orders
| Column | Type | Notes |
|---|---|---|
| id | varchar(14) PK | 14-char hex, crypto/rand |
| user_id | varchar(36), indexed | |
| product_id | varchar(36) | |
| amount | bigint | computed = product.amount × quantity |
| quantity | int | |
| status | varchar(20) | `success` / `failed` |
| metadata | json | all keys from request `metadata` + `latitude`, `longitude`, `quantity` |
| failure_reason | varchar(500) nullable | set when status=failed |
| request_id | varchar(255), **UNIQUE** | `"pay_" + Idempotency-Key` — payment correlation + idempotency token |
| created_at | bigint | epoch ms |

**No separate `idempotency_keys` table.** Idempotency is enforced via the UNIQUE index on `orders.request_id`. A duplicate `Idempotency-Key` resolves to the existing order in the lookup path.

### Seed Data

5 users (3 active, 1 inactive, 1 created) + 10 products. Auto-seeded on `docker compose up`.

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

Active users for testing: `user-001` (Alice), `user-002` (Bob), `user-004` (Diana).
Inactive: `user-003` (Charlie). Not-yet-active: `user-005` (Eve).

### APIs

| Method | Path | Description |
|---|---|---|
| GET | `/v1/products?product_name=Lays` | Search products by name (substring match on description + brand) |
| GET | `/v1/products/{product_id}` | Get product by ID |
| POST | `/v1/orders` | Place order (requires `Idempotency-Key` header) |

#### POST /v1/orders — Request

Header: `Idempotency-Key: <unique-per-attempt>` (required)

All body fields are **mandatory**:

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

- `metadata` is a **key-value map** (`map[string]string`). Must contain a non-empty `address` key. Any extra keys are persisted as-is on the order.
- `amount` is validated against `product.amount × quantity`. Mismatch → 400.

#### POST /v1/orders — Response

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

On payment failure: HTTP 200, `status: "failed"`, `failure_reason: "<reason>"`, stock untouched.

## Run

### Docker Compose (recommended)

```bash
docker compose up --build
```

App at `http://localhost:8080`. MySQL auto-starts, app auto-migrates and seeds sample data.

> ⚠️ **Schema changed?** If you previously ran an older version locally, drop the old `orders` table (or the whole DB) before starting — the new UNIQUE index on `request_id` will fail on legacy duplicate rows:
>
> ```bash
> docker compose down -v   # wipes the mysql volume
> # OR, against a manually-managed MySQL:
> mysql -u root -p -e "DROP DATABASE IF EXISTS quickcommerce;"
> ```

### Local (needs MySQL running)

```bash
# 1. Create DB and run migrations
mysql -u root -p < migrations/001_create_tables.sql
mysql -u root -p < migrations/002_seed_data.sql

# 2. Start server
export DB_HOST=localhost DB_PORT=3306 DB_USER=root DB_PASSWORD=password DB_NAME=quickcommerce
go run ./cmd/server
```

### Migrations only (no app)

```bash
mysql -u root -p < migrations/001_create_tables.sql   # schema
mysql -u root -p < migrations/002_seed_data.sql        # seed users + products
```

Note: `docker compose up` auto-migrates + seeds via GORM AutoMigrate. SQL migrations provided for manual/standalone MySQL setups.

## Test

### Unit + E2E (needs Docker daemon)

```bash
go test ./... -v -count=1
```

E2E tests use [dockertest](https://github.com/ory/dockertest) — spins up a real MySQL container, seeds data, fires 20 concurrent orders against stock=5, asserts exactly 5 succeed and stock=0.

### Manual API Testing

- **VS Code / JetBrains:** Open `api/quick-commerce.http` and run requests in order.
- **Postman:** Import `api/quick-commerce.postman_collection.json`.

Flow: Search → Get Product → Place Order → Replay (idempotency) → Out of Stock → Missing Key (400) → Inactive User.

## Curl Examples

```bash
# Search products
curl http://localhost:8080/v1/products?product_name=Lays

# Get product by ID
curl http://localhost:8080/v1/products/prod-001

# Place order
curl -X POST http://localhost:8080/v1/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: my-unique-key-001" \
  -d '{
    "user_id": "user-001",
    "product_id": "prod-001",
    "quantity": 1,
    "amount": 2000,
    "metadata": { "address": "123 Main St, Bangalore" },
    "latitude": "12.9716",
    "longitude": "77.5946"
  }'

# Replay same key (returns same order, no double charge)
curl -X POST http://localhost:8080/v1/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: my-unique-key-001" \
  -d '{
    "user_id": "user-001",
    "product_id": "prod-001",
    "quantity": 1,
    "amount": 2000,
    "metadata": { "address": "123 Main St, Bangalore" },
    "latitude": "12.9716",
    "longitude": "77.5946"
  }'
```

## Validation Rules (POST /v1/orders)

All of the following must be present and non-empty (else HTTP 400):
- `Idempotency-Key` header
- `user_id`, `product_id`
- `quantity` > 0
- `amount` > 0 (and must equal `product.amount × quantity`)
- `latitude`, `longitude`
- `metadata` (non-empty map) — must contain a non-empty `address` key

## Deviations from Original Discussion

1. **Money as integer paise** — spec showed plain amounts; paise avoids float rounding.
2. **MySQL row lock vs in-process mutex** — "mutex keyed by product_id" implemented as `SELECT ... FOR UPDATE` row lock so consistency holds across multiple app instances (horizontally scalable).
3. **Server recomputes amount** — client `amount` is validated against `product.amount × quantity`; server is source of truth.
4. **Payment failure returns HTTP 200** — with `status: "failed"` and `failure_reason` in body, so client sees the reason. Not a 5xx.
5. **Timestamps as epoch milliseconds** — not SQL datetime; all `created_at`/`updated_at` are `int64` Unix millis.
6. **Order ID is 14-char hex** — short, URL-safe, generated via `crypto/rand`. Not UUID.
7. **No separate `idempotency_keys` table** — `Idempotency-Key` header is stored as `orders.request_id = "pay_" + key` (UNIQUE indexed). Single source of truth; replay path = lookup-by-request_id.
8. **All order fields mandatory** — `user_id`, `product_id`, `quantity`, `amount`, `latitude`, `longitude`, `metadata.address` are all required.
9. **`metadata` is a key-value map** — `map[string]string`, must contain `address`. Extra keys are persisted.

## Project Structure

```
cmd/server/main.go           — entrypoint, wiring, migrate, seed, graceful shutdown
internal/config/              — env-based config
internal/domain/              — entities, status enums, domain errors
internal/payment/             — Payment interface + mock (always success)
internal/repository/          — GORM repos + TX manager (ctx-scoped)
internal/service/             — business logic (OrderService, ProductService)
internal/handler/             — HTTP handlers, DTOs, chi router
internal/httperr/             — domain error → HTTP status mapping
test/e2e/                     — integration tests (dockertest + real MySQL)
api/                          — .http + Postman collection for manual flow testing
```
