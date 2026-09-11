# ecommerce-sas

A multi-tenant e-commerce SaaS: sellers subscribe, spin up a store, list products,
and sell them; shoppers browse, add to a cart, check out, and pay by **card
(Stripe)** or **cash on delivery**.

It is a Go microservice monorepo — 8 services, one PostgreSQL database each,
fronted by nginx, wired together over HTTP (a shared JWT for user auth, a shared
`X-Internal-Key` for service-to-service calls).

| Service | Port | Owns |
|---|---|---|
| **user-management** | 8081 | accounts, auth (password + Google/Facebook OAuth, refresh tokens in Redis), tenants (stores), memberships, roles, permissions |
| **product-service** | 8082 | catalog — products, variants (price + SKU), images (S3 / MinIO), category tree, **per-category attributes** (VARIANT axes / SPEC fields), **full-text search**, **customer reviews** |
| **inventory-service** | 8083 | stock levels per SKU (on-hand / reserved / available) + an append-only movement ledger |
| **cart-service** | 8084 | one active shopping cart per shopper per store; snapshots prices from product-service |
| **order-service** | 8085 | checkout orchestrator — addresses, **shipping / tax / coupons**, idempotency, reserves stock, drives payment & fulfilment (tracking), **returns / RMA** |
| **payment-service** | 8086 | order payments (Stripe card + cash-on-delivery) **and** seller subscriptions/plans |
| **mail-service** | 8087 | transactional email — templates + SMTP relay; consumes order/payment events (no public API) |
| **notification-service** | 8088 | the shopper's in-app notification feed; consumes order/payment events |

The browser only ever talks to **nginx on `:80`**; ports 8081-8088 are for debugging.

Each service has its own `services/<name>/README.md` with the full API. This file
is the **platform** view: how to run everything, what third-party keys to supply,
and the end-to-end business flow to follow (and test) in order.

---

## 1. Run the whole thing (dev)

Prerequisites: **Docker + Docker Compose**. Nothing else — Go, Postgres, Kafka,
Redis and MinIO all run in containers.

```bash
# from the repo root
docker compose --env-file .env.dev -f deployments/compose/docker-compose.dev.yml up -d --build
```

This starts: postgres, redis, kafka, minio (+ bucket setup), nginx, and all 8
services (hot-reloading via `air` — your local source is bind-mounted).

### Apply each database schema (one time)

No service auto-migrates. Run each migrator once:

```bash
for s in user-management product-service inventory-service cart-service \
         order-service payment-service mail-service notification-service; do
  docker exec "$s" sh -c 'cd /app && go run ./cmd/migration up'
done
```

(`... go run ./cmd/migration status` shows applied/pending; `... down` rolls back one.)

### Check it's alive

```bash
curl -s http://localhost:8081/api/v1/health   # user-management (repeat for 8082-8088)
curl -s http://localhost/health               # through nginx
```

Everything the browser touches goes through **nginx on :80** (`http://localhost`).
The per-service ports (8081-8088) are exposed only for debugging.

### Stop / reset

```bash
docker compose -f deployments/compose/docker-compose.dev.yml down          # stop
docker compose -f deployments/compose/docker-compose.dev.yml down -v       # stop + wipe data
```

---

## 2. Third-party configuration

Everything runs out of the box in **stub mode** — no external accounts needed.
Card payments "succeed" instantly and emails are logged instead of sent. Supply
real keys in `.env.dev` when you want the real thing.

| Feature | Env vars (in `.env.dev`) | Without it |
|---|---|---|
| **Stripe** (card payments) | `PAYMENT_STRIPE_SECRET_KEY` (`sk_test_…`), `PAYMENT_STRIPE_WEBHOOK_SECRET` (`whsec_…`) | Stub gateway — CARD payments capture immediately, no real charge |
| **SMTP** (outbound email) | `MAIL_SMTP_HOST`, `MAIL_SMTP_PORT`, `MAIL_SMTP_USERNAME`, `MAIL_SMTP_PASSWORD`, `MAIL_SMTP_FROM` | Dev delivers to the bundled **Mailpit** container — inbox UI at `http://localhost:8025`. Blank `MAIL_SMTP_HOST` = stub (mail written to the `mails` table + log). For real email, swap in a provider's SMTP creds (Resend / Postmark / Brevo / SES / Gmail app password) |
| **Google login** | `USER_AUTH_GOOGLE_CLIENT_ID`, `USER_AUTH_GOOGLE_CLIENT_SECRET` | The `/auth/google` routes 400; email + password still works |
| **Facebook login** | `USER_AUTH_FACEBOOK_CLIENT_ID`, `USER_AUTH_FACEBOOK_CLIENT_SECRET` | The `/auth/facebook` routes 400; email + password still works |
| **AWS S3** (product images in prod) | `PRODUCT_S3_*` (bucket, region, keys) — leave `PRODUCT_S3_ENDPOINT` empty for real AWS | Dev uses the bundled **MinIO** container as an S3 stand-in |

### Stripe test setup (optional)

1. Create a [Stripe](https://dashboard.stripe.com) account, stay in **test mode**.
2. Put the **Secret key** in `PAYMENT_STRIPE_SECRET_KEY`.
3. For webhooks, run `stripe listen --forward-to localhost:8086/api/v1/webhooks/stripe`
   and copy the `whsec_…` it prints into `PAYMENT_STRIPE_WEBHOOK_SECRET`.
4. `docker compose … up -d payment-service` to reload.

Card flow with real Stripe: `POST /orders/:id/pay` returns a `client_secret`; the
browser confirms it with Stripe.js; `POST /orders/:id/pay` again (or the webhook)
moves the order to `CONFIRMED`.

### Shared secrets

`.env.dev` uses `change-this-in-development` for every `*_JWT_SECRET` (they **must
match** — every service verifies user-management's token) and `dev-internal-key`
for every `*_INTERNAL_KEY` (service-to-service). Change both sets for any real
deployment.

---

## 3. The business flow — in order

Run these against `http://localhost` (nginx). Auth is a cookie (`access_token`)
set on login; the examples use `-b/-c cookies.txt` to carry it. `TENANT`, `PID`,
`SKU`, `OID` are ids you capture from earlier responses.

### 3.1 A seller signs up and subscribes

```bash
BASE=http://localhost/api/v1

# register (or POST /auth/login if you already have an account)
curl -sc cookies.txt -X POST $BASE/auth/register -H 'Content-Type: application/json' \
  -d '{"name":"Sam Seller","email":"sam@shop.test","password":"s3cret-pass"}'

# a store needs an active subscription — see the plans, then subscribe
curl -sb cookies.txt $BASE/plans
curl -sb cookies.txt -X POST $BASE/subscription -H 'Content-Type: application/json' \
  -d '{"plan_code":"starter-monthly"}'     # or starter-yearly ($25/mo billed yearly)
```

> Subscriptions are **per user**. `starter-monthly` bills every month; `starter-yearly`
> bills once a year at a lower per-month rate. Subscribing is a stub charge in
> payment-service (no card yet) — it just records an ACTIVE entitlement.

### 3.2 Create the store

```bash
curl -sb cookies.txt -X POST $BASE/tenants -H 'Content-Type: application/json' \
  -d '{"name":"Sam'\''s Shop","slug":"sams-shop"}'
# -> capture data.id as TENANT. Sam is auto-enrolled as this tenant's ADMIN.
```

user-management calls payment-service here to confirm the subscription. No active
plan → `402`-style `VALIDATION_ERROR` ("an active subscription is required").

### 3.3 Build the catalog (store ADMIN / MANAGER)

```bash
# category
curl -sb cookies.txt -X POST $BASE/tenants/$TENANT/categories -H 'Content-Type: application/json' \
  -d '{"name":"Laptops","slug":"laptops"}'

# product (starts as DRAFT)
curl -sb cookies.txt -X POST $BASE/tenants/$TENANT/products -H 'Content-Type: application/json' \
  -d '{"name":"ThinkPad X1","slug":"thinkpad-x1","description":"14-inch ultrabook"}'
# -> capture data.id as PID

# a variant — this is where price + SKU live
curl -sb cookies.txt -X POST $BASE/tenants/$TENANT/products/$PID/variants -H 'Content-Type: application/json' \
  -d '{"sku":"TPX1-16-512","price_cents":149900,"currency":"USD","is_default":true}'

# (optional) image: ask for a presigned URL, PUT the file to it, then confirm.
#   POST $BASE/tenants/$TENANT/products/$PID/images/upload-url  -> {upload_url, storage_key}
#   curl -X PUT --upload-file photo.jpg -H 'Content-Type: image/jpeg' "<upload_url>"
#   POST $BASE/tenants/$TENANT/products/$PID/images  -d '{"storage_key":"..."}'
```

**Category attributes — selectable options vs. display specs.** Every category can
define fields that its products inherit. Each field is either a **VARIANT** axis
(the customer picks it; each combination is a priced variant — "12GB" costs more
than "8GB") or a **SPEC** (display only — "Chipset", "Battery").

```bash
CID=<category id>
# a VARIANT axis with its choices
RAM=$(curl -sb cookies.txt -X POST $BASE/tenants/$TENANT/categories/$CID/attributes \
  -H 'Content-Type: application/json' \
  -d '{"name":"RAM","code":"ram","role":"VARIANT"}' | jq -r .data.id)
V8=$(curl -sb cookies.txt -X POST  $BASE/tenants/$TENANT/categories/$CID/attributes/$RAM/values \
  -H 'Content-Type: application/json' -d '{"value":"8GB"}'  | jq -r .data.id)
V12=$(curl -sb cookies.txt -X POST $BASE/tenants/$TENANT/categories/$CID/attributes/$RAM/values \
  -H 'Content-Type: application/json' -d '{"value":"12GB"}' | jq -r .data.id)

# a SPEC field (display only)
CHIP=$(curl -sb cookies.txt -X POST $BASE/tenants/$TENANT/categories/$CID/attributes \
  -H 'Content-Type: application/json' \
  -d '{"name":"Chipset","code":"chipset","role":"SPEC"}' | jq -r .data.id)

# put the product in the category, then...
curl -sb cookies.txt -X PUT $BASE/tenants/$TENANT/products/$PID/categories \
  -H 'Content-Type: application/json' -d "{\"category_ids\":[\"$CID\"]}"

# ...give it its display-spec values
curl -sb cookies.txt -X PUT $BASE/tenants/$TENANT/products/$PID/specs \
  -H 'Content-Type: application/json' \
  -d "{\"specs\":[{\"attribute_id\":\"$CHIP\",\"value\":\"Snapdragon 8 Gen 3\"}]}"

# ...and make each variant a specific combination (its own price)
curl -sb cookies.txt -X POST $BASE/tenants/$TENANT/products/$PID/variants \
  -H 'Content-Type: application/json' \
  -d "{\"sku\":\"TPX1-8\",\"price_cents\":149900,\"options\":[{\"attribute_id\":\"$RAM\",\"value_id\":\"$V8\"}]}"
curl -sb cookies.txt -X POST $BASE/tenants/$TENANT/products/$PID/variants \
  -H 'Content-Type: application/json' \
  -d "{\"sku\":\"TPX1-12\",\"price_cents\":169900,\"options\":[{\"attribute_id\":\"$RAM\",\"value_id\":\"$V12\"}]}"
```

The public product page (`GET /tenants/$TENANT/catalog/$PID`) then returns
`options` (the axes + choices to render selectors), `specs` (the display lines),
and each entry in `variants[]` carries `options: {"ram":"12GB"}` — so the
storefront maps a selection → variant → price.

### 3.4 Stock the shelves + configure checkout (ADMIN / MANAGER)

```bash
SKU=TPX1-16-512
# register the SKU for tracking, then receive 50 units
curl -sb cookies.txt -X POST $BASE/tenants/$TENANT/inventory -H 'Content-Type: application/json' \
  -d '{"sku":"'"$SKU"'","reorder_level":5}'
curl -sb cookies.txt -X POST $BASE/tenants/$TENANT/inventory/$SKU/receive -H 'Content-Type: application/json' \
  -d '{"quantity":50,"reference":"PO-1001"}'

# a shipping rate (checkout needs one) — free over $100
curl -sb cookies.txt -X POST $BASE/tenants/$TENANT/shipping-rates -H 'Content-Type: application/json' \
  -d '{"name":"Standard","amount_cents":900,"free_over_cents":10000}'   # -> capture data.id as RATE

# 8.75% sales tax
curl -sb cookies.txt -X PUT $BASE/tenants/$TENANT/tax-config -H 'Content-Type: application/json' \
  -d '{"tax_rate_bps":875,"tax_inclusive":false}'

# a 10%-off coupon
curl -sb cookies.txt -X POST $BASE/tenants/$TENANT/coupons -H 'Content-Type: application/json' \
  -d '{"code":"WELCOME10","kind":"PERCENT","value":10,"min_order_cents":5000}'
```

> A SKU that inventory-service doesn't know about is treated as **unlimited** by
> the cart. A tenant with no tax-config pays no tax; with no shipping rate, checkout
> can't proceed (it needs `shipping_rate_id`).

### 3.5 Publish the product

```bash
curl -sb cookies.txt -X PATCH $BASE/tenants/$TENANT/products/$PID -H 'Content-Type: application/json' \
  -d '{"status":"ACTIVE"}'     # needs at least one variant

curl -s $BASE/tenants/$TENANT/catalog          # public — now shows the product
curl -s $BASE/tenants/$TENANT/stock/$SKU        # public — { available: 50, in_stock: true }
```

### 3.6 A shopper buys (new account)

```bash
curl -sc buyer.txt -X POST $BASE/auth/register -H 'Content-Type: application/json' \
  -d '{"name":"Bea Buyer","email":"bea@mail.test","password":"buyer-pass-1"}'

# add to cart (product_id + sku + qty). The cart snapshots the price.
curl -sb buyer.txt -X POST $BASE/tenants/$TENANT/cart/items -H 'Content-Type: application/json' \
  -d '{"product_id":"'"$PID"'","sku":"'"$SKU"'","quantity":2}'
curl -sb buyer.txt $BASE/tenants/$TENANT/cart

# (optional) preview a coupon against the cart subtotal
curl -sb buyer.txt -X POST $BASE/tenants/$TENANT/coupon-check -H 'Content-Type: application/json' \
  -d '{"code":"WELCOME10","subtotal_cents":299800}'

# checkout: shipping address + a shipping rate are required; coupon optional.
# The Idempotency-Key header makes a retried POST return the same order.
curl -sb buyer.txt -X POST $BASE/tenants/$TENANT/orders \
  -H 'Content-Type: application/json' -H 'Idempotency-Key: chk-001' \
  -d '{
    "shipping_address": {"recipient_name":"Bea Buyer","phone":"+15550100",
      "address_line_1":"1 Main St","city":"Springfield","country":"US"},
    "shipping_rate_id": "'"$RATE"'",
    "coupon_code": "WELCOME10"
  }'
# -> order PENDING_PAYMENT, stock reserved; data.grand_total_cents already includes
#    subtotal - discount + shipping + tax. Capture data.id as OID.
```

### 3.7 Pay

```bash
# cash on delivery — order goes straight to CONFIRMED, cash collected on delivery
curl -sb buyer.txt -X POST $BASE/tenants/$TENANT/orders/$OID/pay -H 'Content-Type: application/json' \
  -d '{"payment_method":"COD"}'

# --- OR --- card. With stub Stripe this captures immediately -> CONFIRMED + PAID.
curl -sb buyer.txt -X POST $BASE/tenants/$TENANT/orders/$OID/pay -H 'Content-Type: application/json' \
  -d '{"payment_method":"CARD"}'
#   real Stripe: the response carries client_secret; finish it in the browser,
#   then call /pay again to re-check.
```

order-service published `order.placed` (and, once paid, `order.confirmed`) to
Kafka; a moment later notification-service has created the in-app notification and
mail-service has sent the confirmation email (logged, in stub mode — check
`docker logs mail-service`):

```bash
curl -sb buyer.txt $BASE/notifications
curl -sb buyer.txt $BASE/notifications/unread-count
curl -sb buyer.txt -X POST $BASE/notifications/read -H 'Content-Type: application/json' -d '{"all":true}'
```

### 3.8 The seller fulfils

```bash
# as Sam (store ADMIN) — ships the reserved stock (+ COD settle); tracking is optional
curl -sb cookies.txt -X POST $BASE/tenants/$TENANT/orders/$OID/fulfil \
  -H 'Content-Type: application/json' -d '{"carrier":"UPS","tracking_number":"1Z999AA10123456784"}'
# order -> FULFILLED, on_hand 50 -> 48, reserved -> 0. The shipped email carries the tracking number.

curl -sb cookies.txt "$BASE/tenants/$TENANT/orders?scope=all&status=CONFIRMED"   # seller's queue
```

### 3.9 Cancellation, or a return

```bash
# before FULFILLED — cancel: releases stock, refunds a captured card payment
curl -sb buyer.txt -X POST $BASE/tenants/$TENANT/orders/$OID/cancel

# after FULFILLED — the shopper requests a return
curl -sb buyer.txt -X POST $BASE/tenants/$TENANT/orders/$OID/returns -H 'Content-Type: application/json' \
  -d '{"reason":"wrong size","items":[{"sku":"'"$SKU"'","quantity":1}]}'   # -> capture data.id as RID

# the seller approves — partial refund (that one unit) + restock
curl -sb cookies.txt -X POST $BASE/tenants/$TENANT/returns/$RID/resolve -H 'Content-Type: application/json' \
  -d '{"approve":true,"restock":true}'
# return -> COMPLETED; payment refunded_cents rises; inventory on_hand +1
```

### 3.10 Reviews & search (any time after 3.5)

```bash
curl -sb buyer.txt -X POST $BASE/tenants/$TENANT/catalog/$PID/reviews -H 'Content-Type: application/json' \
  -d '{"rating":5,"title":"Great laptop","body":"Fast and light."}'
curl -s $BASE/tenants/$TENANT/catalog/$PID/reviews          # public: { summary:{average,count}, items:[...] }
curl -s "$BASE/tenants/$TENANT/catalog?q=ultrabook"         # full-text search over name + description
```

### State machines

```
order.status:     PENDING_PAYMENT ──pay──▶ CONFIRMED ──fulfil──▶ FULFILLED
                        │                      │
                        └──────── cancel ──────┴──▶ CANCELLED

order.payment_status:  PENDING ──▶ PAID ──▶ REFUNDED
  CARD: PAID on Stripe capture.   COD: PAID when the order is fulfilled (cash in hand).

order money:  grand_total = subtotal − discount + shipping + tax   (frozen at checkout)

return.status:  REQUESTED ──resolve──▶ COMPLETED (refund ± restock)  |  REJECTED

payment.status:  PENDING ──▶ CAPTURED ──▶ REFUNDED   (or FAILED)
  partial refund (returns): stays CAPTURED, refunded_cents rises

inventory movement per order:  checkout=RESERVE, fulfil=SHIP, cancel=RELEASE, return=RESTOCK(→RECEIVE)
```

---

## 4. Who calls whom

### Synchronous — HTTP, in the request path (must succeed)

| Caller | Callee | Why | Auth |
|---|---|---|---|
| every service | user-management `/me/memberships` | resolve the caller's tenant role | user's Bearer token |
| cart-service | product-service `/catalog/:id` | snapshot variant price + name | (public) |
| cart-service | inventory-service `/stock/:sku` | availability check on add | (public) |
| order-service | cart-service `/cart` | read + clear the cart at checkout | user's Bearer token |
| order-service | inventory-service `/internal/…/stock/{reserve,release,ship,restock}` | move stock for the whole order in one tx | `X-Internal-Key` |
| order-service | payment-service `/internal/…/payments…` | create / capture / settle / refund (full or partial) | `X-Internal-Key` |
| user-management | payment-service `/internal/users/:id/subscription` | gate store creation | `X-Internal-Key` |
| payment-service | Stripe API | card PaymentIntents + refunds | Stripe secret key |
| mail-service | user-management `/internal/users/:id` | resolve a recipient's name + email | `X-Internal-Key` |
| mail-service | SMTP relay | deliver the email | SMTP creds |

`/api/v1/internal/**` is **never** proxied by nginx — reachable only inside the
Docker network, and only with the key.

### Asynchronous — Kafka events (durable, decoupled)

Publishers never wait on consumers; a consumer that's down catches up when it
returns (consumer groups + committed offsets).

| Publisher | Topic | Events |
|---|---|---|
| order-service | `order-events` | `order.placed`, `order.confirmed`, `order.shipped`, `order.cancelled` |
| payment-service | `payment-events` | `payment.captured`, `payment.refunded` |

| Consumer (group) | Reads | Does |
|---|---|---|
| **notification-service** | both topics | creates the shopper's in-app notification |
| **mail-service** | both topics | resolves the recipient, renders + sends the matching email |

Every message is a JSON envelope `{ id, type, occurred_at, data }`. Handlers are
**at-least-once**: a failing handler is retried a few times, then logged and
skipped so one bad message can't stall the group. (A transactional outbox would
close the last gap — a crash between the DB commit and the publish — and is the
natural next step.)

---

## 5. Repo layout

```
services/<name>/           one Go module per service; each has its own README
  cmd/api                  the HTTP server
  cmd/migration            forward/backward SQL migrator (up | down [n] | status)
  configs/                 config.yaml + config.<env>.yaml (env vars override)
  internal/
    platform httpx middleware config server   shared infra (near-identical across services)
    <feature>/                                domain / dto / repository / services / handler / routes
    <name>client                              typed HTTP client for another service
  migrations/               000001_… .up.sql / .down.sql
deployments/
  compose/docker-compose.dev.yml   the dev stack
  nginx/nginx.conf                 the edge router
infrastructure/postgres/init/      creates all 8 databases on first boot
.env.dev / .env.prod / .env.test   one env file per environment
go.work                            ties the modules together for local dev
```

### Conventions

- **Response envelope:** `{ "status": "success", "data": … }`; lists wrap as
  `data: { items: [...], limit, offset }`.
- **Error envelope:** `{ "status": "error", "error": { service, code, message,
  method, path, request_id, timestamp, fields[] } }` — same shape everywhere.
- **No N+1 queries.** List endpoints that include children fetch them in one
  `WHERE parent_id = ANY($ids)` query. Bulk writes are one multi-row statement.
- **Money** is integer cents. **Time** is `TIMESTAMPTZ`, UTC.
- **Sync vs async.** Anything transactional (reserve stock, charge a card, check
  a role) is a direct HTTP call. Notifications and side-effects are Kafka events —
  `internal/events` is the shared producer/consumer helper (a `kafka.Writer` +
  a consumer-group `kafka.Reader`), copied into each service.
- Cross-service references (`tenant_id`, `sku`, `order_id`, …) are plain columns —
  no foreign keys across service boundaries.

---

## 6. Tests

Every service has a router smoke test (registers all routes, no DB):

```bash
cd services/<name> && go test ./...
```

For an end-to-end check, bring the stack up (section 1) and walk section 3.
