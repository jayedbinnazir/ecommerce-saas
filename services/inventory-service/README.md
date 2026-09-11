# inventory-service

Tracks **stock levels per SKU per tenant** — on-hand, reserved, available — and
keeps an append-only **movement ledger** of every change. It does *not* own the
catalog (that is product-service); the SKU is the shared key. There is no
cross-service foreign key — `tenant_id` and `sku` are plain columns.

- **Stack:** Go + Gin, PostgreSQL (`database/sql` + `lib/pq`)
- **Base URL (dev):** `http://localhost:8083/api/v1`
- **Module path:** `github.com/jayedbinnazir/inventory-service`

---

## Running it

```bash
# from repo root: brings up postgres, kafka, user-management, product-service, inventory-service
docker compose --env-file .env.dev -f deployments/compose/docker-compose.dev.yml up -d

# apply the schema (no auto-migrate yet)
docker exec inventory-service sh -c 'cd /app && go run ./cmd/migration up'
```

Locally without Docker: `APP_ENV=dev go run ./cmd/migration up && APP_ENV=dev go run ./cmd/api`.

### Config (env, `INVENTORY_` prefix)

| key | notes |
|---|---|
| `INVENTORY_JWT_SECRET` | **must equal user-management's `USER_JWT_SECRET`** — tokens are verified locally |
| `INVENTORY_USER_SERVICE_URL` | e.g. `http://user-management:8080` — used to resolve tenant membership |
| `INVENTORY_DB_*`, `POSTGRES_USER/PASSWORD` | database `inventory_db` |

---

## How auth works

Same model as product-service. inventory-service does **not** own sessions. It:

1. **Verifies the access token locally** with the shared secret (signature + expiry).
   Send it as the `access_token` cookie or `Authorization: Bearer <token>`.
2. **Resolves tenant role** for management routes by calling user-management
   `GET /api/v1/me/memberships` with your token (cached ~30s). A super-admin
   bypasses the check.

| Access | Meaning |
|---|---|
| 🔓 public | no token needed |
| 🏪 | tenant `ADMIN` or `MANAGER` (or platform super-admin) |
| 🔒 internal | service-to-service — `X-Internal-Key: <INVENTORY_INTERNAL_KEY>` header, no user token |

The per-SKU `reserve` / `release` / `ship` routes are 🏪 (manual admin use).
order-service uses the **bulk** `🔒 internal` variants below at checkout, so a
customer never needs stock-write permission. Set `INVENTORY_INTERNAL_KEY` to
enable them (empty = disabled).

---

## Response & error envelopes

Success: `{ "status": "success", "data": ... }`. Paginated lists wrap data as
`{ "items": [...], "limit": 20, "offset": 0 }`. `204` has an empty body.

Errors are self-describing (same contract as the other services):

```json
{
  "status": "error",
  "error": {
    "service": "inventory-service",
    "code": "VALIDATION_ERROR",
    "message": "not enough available stock",
    "method": "POST",
    "path": "/api/v1/tenants/…/inventory/TPX1-16-512/reserve",
    "request_id": "…",
    "timestamp": "2026-09-09T12:00:00Z",
    "fields": []
  }
}
```

| `code` | HTTP | when |
|---|---|---|
| `BAD_REQUEST` | 400 | bad path/query param, non-JSON body |
| `UNAUTHORIZED` | 401 | missing/expired token |
| `FORBIDDEN` | 403 | not a member / wrong role |
| `NOT_FOUND` | 404 | no stock item for that SKU |
| `CONFLICT` | 409 | a stock item for that SKU already exists |
| `VALIDATION_ERROR` | 422 | field validation or a stock rule (insufficient stock, would go negative, …) |
| `UPSTREAM_ERROR` | 502 | user-management unreachable during an authz check |
| `RATE_LIMITED` | 429 | too many requests from one IP |

---

## Model

A **stock item** is one row per `(tenant_id, sku)`:

| field | meaning |
|---|---|
| `on_hand` | units physically in the warehouse |
| `reserved` | units allocated to open orders/carts |
| `available` | `on_hand - reserved` (computed) |
| `reorder_level` | low-stock threshold; `0` disables the flag |
| `low_stock` | `reorder_level > 0 && available <= reorder_level` |

Invariant: `reserved <= on_hand` at all times.

Every quantity change writes one **movement** (`RECEIVE`, `SHIP`, `ADJUST`,
`RESERVE`, `RELEASE`) with a snapshot of `on_hand` / `reserved` afterwards.

| operation | effect |
|---|---|
| `receive` | `on_hand += qty` |
| `adjust` | `on_hand += qty` (qty may be negative; cannot drop below `0` or below `reserved`) |
| `reserve` | `reserved += qty` (fails if `available < qty`) |
| `release` | `reserved -= qty` (fails if `reserved < qty`) |
| `ship` | `on_hand -= qty` **and** `reserved -= qty` (fulfils a reservation) |

---

## Endpoints

| Method | Path | Access |
|---|---|---|
| GET | `/health` | 🔓 |
| GET | `/tenants/{tenantId}/stock/{sku}` | 🔓 — `{ sku, available, in_stock }` only |
| GET | `/tenants/{tenantId}/inventory` | 🏪 — list items, `?q= &low_stock=true &limit= &offset=` |
| POST | `/tenants/{tenantId}/inventory` | 🏪 — register a SKU |
| GET | `/tenants/{tenantId}/inventory/{sku}` | 🏪 — full item |
| PATCH | `/tenants/{tenantId}/inventory/{sku}` | 🏪 — `reorder_level` / `location` |
| DELETE | `/tenants/{tenantId}/inventory/{sku}` | 🏪 |
| POST | `/tenants/{tenantId}/inventory/{sku}/receive` | 🏪 |
| POST | `/tenants/{tenantId}/inventory/{sku}/adjust` | 🏪 |
| POST | `/tenants/{tenantId}/inventory/{sku}/reserve` | 🏪 |
| POST | `/tenants/{tenantId}/inventory/{sku}/release` | 🏪 |
| POST | `/tenants/{tenantId}/inventory/{sku}/ship` | 🏪 |
| GET | `/tenants/{tenantId}/inventory/{sku}/movements` | 🏪 — the ledger, `?limit= &offset=` |
| POST | `/internal/tenants/{tenantId}/stock/reserve` | 🔒 — bulk reserve |
| POST | `/internal/tenants/{tenantId}/stock/release` | 🔒 — bulk release |
| POST | `/internal/tenants/{tenantId}/stock/ship` | 🔒 — bulk ship |

### Bulk stock ops (🔒 internal)

```
POST /api/v1/internal/tenants/{tenantId}/stock/reserve
X-Internal-Key: <key>

{ "reference": "<order id>", "items": [ { "sku": "TPX1-16-512", "quantity": 2 } ] }
```

All lines are applied in **one transaction** (rows locked in SKU order) — every
line succeeds or none do. `release` and `ship` take the same body. A shortfall
returns `422` naming the SKU; the order-service reads that message.

---

## Stock items

### `POST /tenants/{tenantId}/inventory` 🏪

```json
{ "sku": "TPX1-16-512", "reorder_level": 5, "location": "A-12" }
```

**201**

```json
{
  "status": "success",
  "data": {
    "id": "…", "tenant_id": "…", "sku": "TPX1-16-512",
    "on_hand": 0, "reserved": 0, "available": 0,
    "reorder_level": 5, "low_stock": false, "location": "A-12",
    "created_at": "…", "updated_at": "…"
  }
}
```

**409** `{ "code": "CONFLICT", "message": "a stock item for that SKU already exists" }`

### `GET /tenants/{tenantId}/inventory/{sku}` 🏪

**200** — the same shape as above. **404** if the SKU is not tracked.

### `PATCH /tenants/{tenantId}/inventory/{sku}` 🏪

```json
{ "reorder_level": 10 }
```

Send `"location": ""` to clear it. **422** if the body is empty.

---

## Operations

### `POST /tenants/{tenantId}/inventory/{sku}/receive` 🏪

```json
{ "quantity": 100, "reason": "PO-4471", "reference": "PO-4471" }
```

**200** — the updated stock item (`on_hand` now `100`, `available` `100`).

### `POST /tenants/{tenantId}/inventory/{sku}/adjust` 🏪

```json
{ "quantity": -3, "reason": "damaged in transit" }
```

**200** — updated item. **422**
`{ "code": "VALIDATION_ERROR", "message": "that change would take on-hand below zero or below what is reserved" }`

### `POST /tenants/{tenantId}/inventory/{sku}/reserve` 🏪

```json
{ "quantity": 2, "reference": "order_9f3c…" }
```

**200** — `reserved` goes up by 2, `available` down by 2.
**422** `{ "code": "VALIDATION_ERROR", "message": "not enough available stock" }`

### `POST /tenants/{tenantId}/inventory/{sku}/release` 🏪

```json
{ "quantity": 2, "reference": "order_9f3c…" }
```

**200** — `reserved` goes back down. **422** if `reserved < quantity`.

### `POST /tenants/{tenantId}/inventory/{sku}/ship` 🏪

```json
{ "quantity": 2, "reference": "order_9f3c…" }
```

**200** — `on_hand` **and** `reserved` both drop by 2 (a reservation is fulfilled).
**422** if there is not enough reserved or on-hand.

### `GET /tenants/{tenantId}/inventory/{sku}/movements` 🏪

**200**

```json
{
  "status": "success",
  "data": {
    "items": [
      { "id": "…", "sku": "TPX1-16-512", "type": "SHIP", "quantity": 2,
        "on_hand_after": 98, "reserved_after": 0,
        "reason": null, "reference": "order_9f3c…", "created_by": "…",
        "created_at": "…" },
      { "id": "…", "sku": "TPX1-16-512", "type": "RESERVE", "quantity": 2,
        "on_hand_after": 100, "reserved_after": 2, "reference": "order_9f3c…",
        "created_at": "…" },
      { "id": "…", "sku": "TPX1-16-512", "type": "RECEIVE", "quantity": 100,
        "on_hand_after": 100, "reserved_after": 0, "reference": "PO-4471",
        "created_at": "…" }
    ],
    "limit": 20, "offset": 0
  }
}
```

---

## Storefront view

### `GET /tenants/{tenantId}/stock/{sku}` 🔓

**200**

```json
{ "status": "success", "data": { "sku": "TPX1-16-512", "available": 96, "in_stock": true } }
```

Never exposes `on_hand` or `reserved`. **404** if the SKU is not tracked.

---

## curl walkthrough

```bash
BASE=http://localhost:8083/api/v1
# TOKEN = an access token from user-management, TENANT = a tenant you ADMIN, SKU from product-service
SKU=TPX1-16-512

curl -s -X POST $BASE/tenants/$TENANT/inventory -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d "{\"sku\":\"$SKU\",\"reorder_level\":5}"

curl -s -X POST $BASE/tenants/$TENANT/inventory/$SKU/receive -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"quantity":100,"reference":"PO-4471"}'

curl -s -X POST $BASE/tenants/$TENANT/inventory/$SKU/reserve -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"quantity":2,"reference":"order_1"}'

curl -s -X POST $BASE/tenants/$TENANT/inventory/$SKU/ship -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"quantity":2,"reference":"order_1"}'

curl -s $BASE/tenants/$TENANT/stock/$SKU        # public -> { available: 98, in_stock: true }
```
