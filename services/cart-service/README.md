# cart-service

One **shopping cart per shopper per tenant**. Lines are keyed by variant `sku`.
When a line is added, cart-service snapshots the price and names from
product-service so the total stays stable, and checks stock against
inventory-service.

- **Stack:** Go + Gin, PostgreSQL (`database/sql` + `lib/pq`)
- **Base URL (dev):** `http://localhost:8084/api/v1`
- **Module path:** `github.com/jayedbinnazir/cart-service`

---

## Running it

```bash
# from repo root: brings up postgres, kafka, user-management, product-service, inventory-service, cart-service
docker compose --env-file .env.dev -f deployments/compose/docker-compose.dev.yml up -d

# apply the schema (no auto-migrate yet)
docker exec cart-service sh -c 'cd /app && go run ./cmd/migration up'
```

Locally without Docker: `APP_ENV=dev go run ./cmd/migration up && APP_ENV=dev go run ./cmd/api`.

### Config (env, `CART_` prefix)

| key | notes |
|---|---|
| `CART_JWT_SECRET` | **must equal user-management's `USER_JWT_SECRET`** — tokens are verified locally |
| `CART_USER_SERVICE_URL` | user-management base URL (used by the shared auth guard) |
| `CART_PRODUCT_SERVICE_URL` | product-service base URL — price/name snapshots |
| `CART_INVENTORY_SERVICE_URL` | inventory-service base URL — stock checks |
| `CART_DB_*`, `POSTGRES_USER/PASSWORD` | database `cart_db` |

---

## How auth works

Every route needs a valid access token (cookie `access_token` or
`Authorization: Bearer <token>`). The cart belongs to whoever the token
identifies (`customer_id`) — **no tenant membership is required**, anyone signed
in can shop in any tenant.

The token is verified locally with the shared secret. cart-service forwards it
to product-service and inventory-service, though those endpoints are public
today.

---

## Downstream calls

| when | call | on failure |
|---|---|---|
| add item | `GET product-service /tenants/{t}/catalog/{productId}` → find variant by `sku` | not found / not ACTIVE → `422 VALIDATION_ERROR` ("not available"); unreachable → `502 UPSTREAM_ERROR` |
| add / increase item | `GET inventory-service /tenants/{t}/stock/{sku}` | `404` (sku not tracked) → treated as unlimited; not enough → `422`; unreachable → `502` |

Price, currency, product name and variant title are **snapshotted** into the cart
line. Re-adding the same sku only changes the quantity — the original price is kept.

---

## Response & error envelopes

Success: `{ "status": "success", "data": ... }`. `POST /items` returns `201`,
everything else `200`. Errors use the shared contract:

```json
{
  "status": "error",
  "error": {
    "service": "cart-service",
    "code": "VALIDATION_ERROR",
    "message": "not enough stock for the requested quantity",
    "method": "POST",
    "path": "/api/v1/tenants/…/cart/items",
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
| `NOT_FOUND` | 404 | the item is not in the cart |
| `VALIDATION_ERROR` | 422 | field validation, product not available, not enough stock, currency mismatch |
| `UPSTREAM_ERROR` | 502 | product-service or inventory-service unreachable |
| `RATE_LIMITED` | 429 | too many requests from one IP |

---

## Endpoints

| Method | Path | Notes |
|---|---|---|
| GET | `/health` | — |
| GET | `/tenants/{tenantId}/cart` | my active cart (created empty on first call) |
| DELETE | `/tenants/{tenantId}/cart` | remove every line (keeps the empty cart) |
| POST | `/tenants/{tenantId}/cart/items` | add a line `{ product_id, sku, quantity }` |
| PATCH | `/tenants/{tenantId}/cart/items/{sku}` | set a line's quantity `{ quantity }` (≥ 1) |
| DELETE | `/tenants/{tenantId}/cart/items/{sku}` | remove one line |

Every response is the **whole cart**:

```json
{
  "status": "success",
  "data": {
    "id": "…", "tenant_id": "…", "customer_id": "…",
    "status": "ACTIVE", "currency": "USD",
    "items": [
      {
        "id": "…", "product_id": "…", "sku": "TPX1-16-512",
        "quantity": 2, "unit_price_cents": 149900, "currency": "USD",
        "product_name": "ThinkPad X1", "variant_title": "16GB / 512GB",
        "subtotal_cents": 299800
      }
    ],
    "item_count": 2,
    "subtotal_cents": 299800,
    "created_at": "…", "updated_at": "…"
  }
}
```

An empty cart has `"items": []`, `"currency": ""`, `"item_count": 0`,
`"subtotal_cents": 0`.

---

## Examples

### `POST /tenants/{tenantId}/cart/items`

```json
{ "product_id": "8f3c…", "sku": "TPX1-16-512", "quantity": 2 }
```

**201** — the whole cart (see above).

**422** the product is not on sale:
`{ "code": "VALIDATION_ERROR", "message": "that product is not available for purchase" }`

**422** not enough stock:
`{ "code": "VALIDATION_ERROR", "message": "not enough stock for the requested quantity" }`

**422** mixing currencies:
`{ "code": "VALIDATION_ERROR", "message": "every item in a cart must use the same currency" }`

### `PATCH /tenants/{tenantId}/cart/items/{sku}`

```json
{ "quantity": 5 }
```

**200** — updated cart. Re-checks stock. **404** if the sku is not in the cart.
To remove a line use `DELETE`, not `quantity: 0`.

---

## curl walkthrough

```bash
BASE=http://localhost:8084/api/v1
# TOKEN from user-management POST /auth/login; TENANT a tenant; PID + SKU from product-service
curl -s $BASE/tenants/$TENANT/cart -H "Authorization: Bearer $TOKEN"

curl -s -X POST $BASE/tenants/$TENANT/cart/items -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"product_id\":\"$PID\",\"sku\":\"$SKU\",\"quantity\":2}"

curl -s -X PATCH $BASE/tenants/$TENANT/cart/items/$SKU -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"quantity":3}'

curl -s -X DELETE $BASE/tenants/$TENANT/cart/items/$SKU -H "Authorization: Bearer $TOKEN"
curl -s -X DELETE $BASE/tenants/$TENANT/cart -H "Authorization: Bearer $TOKEN"
```
