# product-service

The store catalog: **products**, their **variants** (price + SKU) and **images**
(stored in S3), plus a per-tenant **category** tree. Stock levels are *not* here —
they belong to inventory-service; other services join on a variant's `sku`.

- **Stack:** Go + Gin, PostgreSQL (`database/sql` + `lib/pq`), AWS S3 (MinIO in dev)
- **Base URL (dev):** `http://localhost:8082/api/v1`
- **Module path:** `github.com/jayedbinnazir/product-service`

---

## Running it

```bash
# from repo root: brings up postgres, kafka, minio, user-management, product-service
docker compose --env-file .env.dev -f deployments/compose/docker-compose.dev.yml up -d

# apply the schema (no auto-migrate yet)
docker exec product-service sh -c 'cd /app && go run ./cmd/migration up'
```

Locally without Docker: `APP_ENV=dev go run ./cmd/migration up && APP_ENV=dev go run ./cmd/api`.

### Config (env, `PRODUCT_` prefix)

| key | notes |
|---|---|
| `PRODUCT_JWT_SECRET` | **must equal user-management's `USER_JWT_SECRET`** — tokens are verified locally |
| `PRODUCT_USER_SERVICE_URL` | e.g. `http://user-management:8080` — used to resolve tenant membership |
| `PRODUCT_DB_*`, `POSTGRES_USER/PASSWORD` | database `product_db` |
| `PRODUCT_S3_BUCKET` / `_REGION` / `_ENDPOINT` / `_PUBLIC_BASE_URL` / `_ACCESS_KEY_ID` / `_SECRET_ACCESS_KEY` | object storage; leave `_ENDPOINT` empty for real AWS |

---

## How auth works

product-service does **not** own sessions. It:

1. **Verifies the access token locally** with the shared secret (signature + expiry).
   Send it as the `access_token` cookie or `Authorization: Bearer <token>`.
2. **Resolves tenant role** for write routes by calling user-management
   `GET /api/v1/me/memberships` with your token (cached ~30s). A super-admin
   bypasses the check.

| Access | Meaning |
|---|---|
| 🔓 public | no token needed |
| 🔑 auth | any valid token |
| 🏪 | tenant `ADMIN` or `MANAGER` (or platform super-admin) |

---

## Response & error envelopes

Success: `{ "status": "success", "data": ... }`. Paginated lists wrap data as
`{ "items": [...], "limit": 20, "offset": 0 }`. `204` has an empty body.

Errors are self-describing (same contract as user-management):

```json
{
  "status": "error",
  "error": {
    "service": "product-service",
    "code": "VALIDATION_ERROR",
    "message": "one or more fields are invalid",
    "method": "POST",
    "path": "/api/v1/tenants/…/products",
    "request_id": "…",
    "timestamp": "2026-09-08T12:00:00Z",
    "fields": [ { "field": "slug", "message": "must be at least 2 characters" } ]
  }
}
```

| `code` | HTTP | when |
|---|---|---|
| `BAD_REQUEST` | 400 | bad path/query param, non-JSON body |
| `UNAUTHORIZED` | 401 | missing/expired token |
| `FORBIDDEN` | 403 | not a member / wrong role |
| `NOT_FOUND` | 404 | no such product / variant / image / category |
| `CONFLICT` | 409 | duplicate slug or SKU |
| `VALIDATION_ERROR` | 422 | field validation or a domain rule (e.g. activating a product with no variants) |
| `UPSTREAM_ERROR` | 502 | user-management unreachable during an authz check |
| `RATE_LIMITED` | 429 | too many requests from one IP |

---

## Endpoints

| Method | Path | Access |
|---|---|---|
| GET | `/health` | 🔓 |
| **Storefront** | | |
| GET | `/tenants/{tenantId}/catalog` | 🔓 — ACTIVE products, `?q= &category= &limit= &offset=` |
| GET | `/tenants/{tenantId}/catalog/{productId}` | 🔓 — full detail, ACTIVE only |
| GET | `/tenants/{tenantId}/categories` | 🔓 |
| GET | `/tenants/{tenantId}/categories/{categoryId}` | 🔓 |
| **Category management** | | |
| POST / PATCH / DELETE | `/tenants/{tenantId}/categories[/{categoryId}]` | 🏪 |
| **Product management** | | |
| GET | `/tenants/{tenantId}/products` | 🏪 — any status, `?status= &q= &category=` |
| POST | `/tenants/{tenantId}/products` | 🏪 |
| GET / PATCH / DELETE | `/tenants/{tenantId}/products/{productId}` | 🏪 |
| PUT | `/tenants/{tenantId}/products/{productId}/categories` | 🏪 |
| POST / PATCH / DELETE | `/tenants/{tenantId}/products/{productId}/variants[/{variantId}]` | 🏪 |
| POST | `/tenants/{tenantId}/products/{productId}/images/upload-url` | 🏪 |
| POST | `/tenants/{tenantId}/products/{productId}/images` | 🏪 |
| PATCH / DELETE | `/tenants/{tenantId}/products/{productId}/images/{imageId}` | 🏪 |

---

## Categories

### `POST /tenants/{tenantId}/categories` 🏪

```json
{ "name": "Laptops", "slug": "laptops", "parent_id": null, "position": 0 }
```

**201**

```json
{
  "status": "success",
  "data": {
    "id": "…", "tenant_id": "…", "parent_id": null,
    "name": "Laptops", "slug": "laptops", "position": 0,
    "created_at": "…", "updated_at": "…"
  }
}
```

**409** `{ "code": "CONFLICT", "message": "a category with that slug already exists" }`

`PATCH` accepts any subset; send `"parent_id": ""` to detach from the parent.

---

## Products

A product is created **DRAFT**. It needs at least one variant before it can be
set to **ACTIVE** (visible in the storefront).

### `POST /tenants/{tenantId}/products` 🏪

```json
{ "name": "ThinkPad X1", "slug": "thinkpad-x1", "description": "14-inch ultrabook" }
```

**201** — returns the full detail (variants/images/category_ids empty for now):

```json
{
  "status": "success",
  "data": {
    "id": "8f3c…", "tenant_id": "…",
    "name": "ThinkPad X1", "slug": "thinkpad-x1", "description": "14-inch ultrabook",
    "status": "DRAFT",
    "created_at": "…", "updated_at": "…",
    "variants": [], "images": [], "category_ids": []
  }
}
```

**409** `{ "code": "CONFLICT", "message": "a product with that slug already exists" }`

### `PATCH /tenants/{tenantId}/products/{productId}` 🏪

```json
{ "status": "ACTIVE" }
```

**422** if the product has no variants:

```json
{ "status": "error", "error": {
  "service": "product-service", "code": "VALIDATION_ERROR",
  "message": "a product needs at least one variant before it can be activated", "…": "…"
}}
```

### `PUT /tenants/{tenantId}/products/{productId}/categories` 🏪

```json
{ "category_ids": ["c1d2…", "e3f4…"] }
```

**200** `{ "status": "success", "data": { "category_ids": ["c1d2…", "e3f4…"] } }`

**422** if any id isn't a category of this tenant:
`{ "code": "VALIDATION_ERROR", "message": "one or more categories do not belong to this tenant" }`

### `GET /tenants/{tenantId}/catalog` 🔓

**200**

```json
{
  "status": "success",
  "data": {
    "items": [
      { "id": "…", "tenant_id": "…", "name": "ThinkPad X1", "slug": "thinkpad-x1",
        "status": "ACTIVE", "created_at": "…", "updated_at": "…" }
    ],
    "limit": 20, "offset": 0
  }
}
```

---

## Variants

### `POST /tenants/{tenantId}/products/{productId}/variants` 🏪

```json
{
  "sku": "TPX1-16-512",
  "title": "16GB / 512GB",
  "price_cents": 149900,
  "currency": "USD",
  "compare_at_price_cents": 179900,
  "weight_grams": 1120,
  "position": 0,
  "is_default": true
}
```

`is_default: true` clears the previous default variant for the product.

**201**

```json
{
  "status": "success",
  "data": {
    "id": "…", "product_id": "…", "sku": "TPX1-16-512", "title": "16GB / 512GB",
    "price_cents": 149900, "currency": "USD", "compare_at_price_cents": 179900,
    "weight_grams": 1120, "position": 0, "is_default": true,
    "created_at": "…", "updated_at": "…"
  }
}
```

**409** `{ "code": "CONFLICT", "message": "a variant with that SKU already exists" }`

`PATCH .../variants/{variantId}` — any subset of the above.
**422** if the variant belongs to a different product:
`{ "code": "VALIDATION_ERROR", "message": "that variant does not belong to this product" }`

---

## Images — direct upload to S3

Two steps: ask for a presigned URL, PUT the file straight to S3, then confirm.
Objects are stored at `products/<tenant_id>/<product_id>/<image_id>.<ext>`.

### 1. `POST /tenants/{tenantId}/products/{productId}/images/upload-url` 🏪

```json
{ "content_type": "image/jpeg" }
```

Allowed: `image/jpeg`, `image/png`, `image/webp`, `image/avif`.

**201**

```json
{
  "status": "success",
  "data": {
    "image_id": "a1b2…",
    "storage_key": "products/<tenant>/<product>/a1b2….jpg",
    "upload_url": "https://…minio…/product-images/products/…?X-Amz-Signature=…",
    "public_url": "http://localhost:9000/product-images/products/…/a1b2….jpg",
    "expires_in": 900
  }
}
```

### 2. Upload the bytes (client → S3, not through this service)

```bash
curl -X PUT --upload-file photo.jpg -H "Content-Type: image/jpeg" "<upload_url>"
```

### 3. `POST /tenants/{tenantId}/products/{productId}/images` 🏪

```json
{ "storage_key": "products/<tenant>/<product>/a1b2….jpg", "alt": "front view", "position": 0 }
```

The service checks the object really exists and that the key is under this
tenant+product's prefix.

**201**

```json
{
  "status": "success",
  "data": {
    "id": "a1b2…", "product_id": "…",
    "url": "http://localhost:9000/product-images/products/…/a1b2….jpg",
    "alt": "front view", "position": 0, "created_at": "…"
  }
}
```

**422** `{ "code": "VALIDATION_ERROR", "message": "no uploaded file found at that storage key" }`
**422** `{ "code": "VALIDATION_ERROR", "message": "storage key does not belong to this product" }`

### `PATCH .../images/{imageId}` — `{ "alt": "...", "position": 2 }` (no URL change)
### `DELETE .../images/{imageId}` — removes the row **and** the S3 object → **204**

---

## curl walkthrough

```bash
BASE=http://localhost:8082/api/v1
# TOKEN = an access token from user-management (POST /auth/login), TENANT = a tenant you ADMIN

curl -s -X POST $BASE/tenants/$TENANT/products -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"ThinkPad X1","slug":"thinkpad-x1"}'                       # -> product id

curl -s -X POST $BASE/tenants/$TENANT/products/$PID/variants -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"sku":"TPX1-16-512","price_cents":149900,"is_default":true}'

curl -s -X PATCH $BASE/tenants/$TENANT/products/$PID -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"status":"ACTIVE"}'

curl -s $BASE/tenants/$TENANT/catalog                                   # public, now shows it
```
