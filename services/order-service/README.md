# order-service

Turns a cart into an **immutable order** and drives it through payment and
fulfilment. It is the checkout orchestrator: it reads cart-service, reserves
stock in inventory-service, writes the order, and clears the cart.

- **Stack:** Go + Gin, PostgreSQL (`database/sql` + `lib/pq`)
- **Base URL (dev):** `http://localhost:8085/api/v1`
- **Module path:** `github.com/jayedbinnazir/order-service`

---

## Running it

```bash
docker compose --env-file .env.dev -f deployments/compose/docker-compose.dev.yml up -d
docker exec order-service sh -c 'cd /app && go run ./cmd/migration up'
```

### Config (env, `ORDER_` prefix)

| key | notes |
|---|---|
| `ORDER_JWT_SECRET` | **must equal user-management's `USER_JWT_SECRET`** |
| `ORDER_USER_SERVICE_URL` | user-management (auth guard + role checks) |
| `ORDER_CART_SERVICE_URL` | cart-service — read + clear the cart at checkout |
| `ORDER_INVENTORY_SERVICE_URL` | inventory-service — reserve / release / ship stock |
| `ORDER_INVENTORY_INTERNAL_KEY` | **must equal `INVENTORY_INTERNAL_KEY`** — auth for the bulk stock API |
| `ORDER_DB_*`, `POSTGRES_USER/PASSWORD` | database `order_db` |

---

## Auth & roles

Every route needs a valid access token. A **customer** acts on their own orders.
A tenant **ADMIN / MANAGER** (checked via user-management `/me/memberships`) can
list every order (`?scope=all`), read any order, and fulfil.

| action | who |
|---|---|
| checkout, pay, cancel, list own, read own | the order's customer |
| list all, read any | ADMIN / MANAGER |
| fulfil | ADMIN / MANAGER |

---

## Order lifecycle

```
          checkout                 pay                  fulfil
  (cart) ──────────▶ PENDING_PAYMENT ──▶ PAID ──────────────▶ FULFILLED
                          │               │
                          └───────┬───────┘
                                cancel
                                  ▼
                              CANCELLED
```

| transition | stock effect (inventory-service) |
|---|---|
| **checkout** | bulk **reserve** every line (`reserved += qty`) |
| **pay** *(stub)* | none — stock stays reserved |
| **fulfil** | bulk **ship** (`on_hand -= qty`, `reserved -= qty`) |
| **cancel** | bulk **release** (`reserved -= qty`) |

Checkout is compensating: if the order row cannot be written after stock was
reserved, the reservation is released before returning the error.

Payment is a **stub** until payment-service (#6) exists — `pay` just records a
fake `payment_ref` and moves the order to `PAID`.

---

## Scale notes

- **No N+1.** Listing orders is always **2 queries**: one for the page of orders,
  one `WHERE order_id = ANY($ids)` for all their items, stitched in memory.
- Checkout writes all line items in **one multi-row INSERT**, inside one
  transaction with the order row.
- Stock changes for a whole order are **one bulk call** to inventory-service
  (`/api/v1/internal/tenants/{t}/stock/{reserve|release|ship}`), applied in a
  single locked transaction there — not one call per SKU.
- Status transitions are conditional `UPDATE ... WHERE id = $1 AND status = $expected`,
  so a double `pay` / `cancel` is a no-op, not a double stock movement.

---

## Response & error envelopes

Success: `{ "status": "success", "data": ... }`. `POST /orders` returns `201`.
Errors use the shared contract (`service: "order-service"`).

| `code` | HTTP | when |
|---|---|---|
| `BAD_REQUEST` | 400 | bad path/query param, non-JSON body |
| `UNAUTHORIZED` | 401 | missing/expired token, or user-management rejected it |
| `FORBIDDEN` | 403 | not your order / not a manager |
| `NOT_FOUND` | 404 | no such order |
| `CONFLICT` | 409 | the order is not in a state that allows this transition |
| `VALIDATION_ERROR` | 422 | empty cart, or inventory refused the reservation (not enough stock) |
| `UPSTREAM_ERROR` | 502 | cart-service / inventory-service / user-management unreachable |

---

## Endpoints

| Method | Path | Who |
|---|---|---|
| GET | `/health` | — |
| POST | `/tenants/{tenantId}/orders` | customer — checkout the active cart |
| GET | `/tenants/{tenantId}/orders` | own orders; `?scope=all&status=PAID&limit=&offset=` for managers |
| GET | `/tenants/{tenantId}/orders/{orderId}` | owner or manager |
| POST | `/tenants/{tenantId}/orders/{orderId}/pay` | owner — `{ "payment_method": "card" }` (stub) |
| POST | `/tenants/{tenantId}/orders/{orderId}/cancel` | owner or manager |
| POST | `/tenants/{tenantId}/orders/{orderId}/fulfil` | manager |

---

## Examples

### `POST /tenants/{tenantId}/orders` — checkout

No body required. Uses the caller's active cart.

**201**

```json
{
  "status": "success",
  "data": {
    "id": "0c1d…", "tenant_id": "…", "customer_id": "…",
    "status": "PENDING_PAYMENT", "currency": "USD",
    "subtotal_cents": 299800, "item_count": 2,
    "items": [
      { "id": "…", "product_id": "…", "sku": "TPX1-16-512", "quantity": 2,
        "unit_price_cents": 149900, "currency": "USD",
        "product_name": "ThinkPad X1", "variant_title": "16GB / 512GB",
        "subtotal_cents": 299800 }
    ],
    "created_at": "…", "updated_at": "…"
  }
}
```

**422** empty cart: `{ "code": "VALIDATION_ERROR", "message": "cart is empty" }`
**422** stock: `{ "code": "VALIDATION_ERROR", "message": "inventory rejected the stock change: not enough available stock" }`

### `POST /tenants/{tenantId}/orders/{orderId}/pay`

```json
{ "payment_method": "card" }
```

**200** — order now `PAID`, `paid_at` set, `payment_ref` populated.
**409** if the order is not `PENDING_PAYMENT`.

### `POST /tenants/{tenantId}/orders/{orderId}/fulfil` (manager)

**200** — order `FULFILLED`; inventory-service ships the reserved units.
**409** if the order is not `PAID`.

### `POST /tenants/{tenantId}/orders/{orderId}/cancel`

**200** — order `CANCELLED`; reserved stock is released.
**409** if the order is already `FULFILLED` or `CANCELLED`.

### `GET /tenants/{tenantId}/orders?scope=all&status=PAID`

**200** — `{ "data": { "items": [ …orders with items… ], "limit": 20, "offset": 0 } }`
(`scope=all` requires ADMIN/MANAGER, else `403`).

---

## curl walkthrough

```bash
BASE=http://localhost:8085/api/v1
# TOKEN from user-management; TENANT a tenant; cart already has items (see cart-service)

OID=$(curl -s -X POST $BASE/tenants/$TENANT/orders -H "Authorization: Bearer $TOKEN" | jq -r .data.id)
curl -s -X POST $BASE/tenants/$TENANT/orders/$OID/pay -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"payment_method":"card"}'
curl -s -X POST $BASE/tenants/$TENANT/orders/$OID/fulfil -H "Authorization: Bearer $MANAGER_TOKEN"
curl -s $BASE/tenants/$TENANT/orders/$OID -H "Authorization: Bearer $TOKEN"
```
