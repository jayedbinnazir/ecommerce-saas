# API Test Results — end-to-end business flow

Live run against the dockerize stack on **2026-09-10**. Every call below was
executed in order; the request body and the **exact** response (success and
error) are copied verbatim.

**Actors**
| Role | How created | Token |
|---|---|---|
| Seller (`Sam Seller`) | `POST /auth/register` | `is_super_admin=false`, becomes tenant **ADMIN** on store creation |
| Customer (`Cara Customer`) | `POST /auth/register` | plain shopper |
| Super admin | seeded (`migration 000010`) | not needed for this flow |

**Base URLs** (host ports)

| Service | URL |
|---|---|
| user-management | `http://localhost:8081` |
| product-service | `http://localhost:8082` |
| inventory-service | `http://localhost:8083` |
| cart-service | `http://localhost:8084` |
| order-service | `http://localhost:8085` |
| payment-service | `http://localhost:8086` |
| notification-service | `http://localhost:8088` |
| Mailpit inbox UI | `http://localhost:8025` |

**IDs produced by this run** (referenced throughout)

```
TENANT   = aa1fad3e-9317-468e-9689-a3bb5c2f7472
CATEGORY = b534729e-cca1-4fe7-9edf-24e13840a554   (Mobiles)
ATTR RAM = eedae797-3c40-4a9f-9f2f-390713a2889c   (VARIANT)
  8GB    = 3fc815ab-d915-4cb2-94c3-f2acd4d60c9d
  12GB   = c56b8f9e-f501-4b25-8719-12c7400d34e3
ATTR CHIP= ae064663-6d9d-4c8b-9f9d-ec69bf7e6e6b   (SPEC)
PRODUCT  = 667f9530-f12a-4e84-b391-d3a2f56180cd   (Galaxy S99)
RATE     = f7adef82-7288-4d3e-bcc9-64c8d4bbc0bb   (Standard shipping)
ORDER    = dc2ca7d7-d379-478f-ba70-9a2432f0f6bd
```

All responses use the envelope `{"status":"success","data":…}` or
`{"status":"error","error":{…}}`.

---

## 0. Health — every service

`GET /api/v1/health` → **200** on all 7 services.

```
user          HTTP 200
product       HTTP 200
inventory     HTTP 200
cart          HTTP 200
order         HTTP 200
payment       HTTP 200
notification  HTTP 200
```

---

## Phase 1 — Seller onboarding

### 1.1 Register the seller — `POST :8081/api/v1/auth/register`

```json
{"name":"Sam Seller","email":"seller_1789037659@shop.test","password":"seller@pass1","phone":"01710000001"}
```
**201**
```json
{"data":{"access_token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9…","refresh_token":"1ld-csWTSnNewKvcnuyHMe_Vwy80OL3wkAK5co56at4","expires_in":900},"status":"success"}
```

### 1.2 Who am I — `GET :8081/api/v1/auth/me`   *(Bearer seller)*

**200**
```json
{"data":{"id":"df883b51-0eb7-4a70-94af-b68a9e1c53d0","name":"Sam Seller","email":"seller_1789037659@shop.test","phone":"01710000001","is_super_admin":false,"created_at":"2026-09-10T10:54:19.93126Z","updated_at":"2026-09-10T10:54:19.93126Z"},"status":"success"}
```

### 1.3 Create a store BEFORE subscribing — `POST :8081/api/v1/tenants`

```json
{"name":"Sam Electronics","slug":"sam-electronics"}
```
**422 — gated by the subscription check**
```json
{"status":"error","error":{"service":"user-management","code":"VALIDATION_ERROR","message":"an active subscription is required to create a store","method":"POST","path":"/api/v1/tenants","request_id":"40cd5420-a9d5-49e3-b8e7-f0250b9e816a","timestamp":"2026-09-10T10:54:21Z"}}
```

### 1.4 List subscription plans — `GET :8086/api/v1/plans`

**200**
```json
{"data":[
  {"id":"44049d78-a540-44bb-a656-a74baff9058a","code":"starter-monthly","name":"Starter — Monthly","interval":"MONTH","price_cents":3000,"monthly_equivalent_cents":3000,"currency":"USD"},
  {"id":"79ebdc3f-1e21-4c28-9a1d-61d16882f328","code":"starter-yearly","name":"Starter — Yearly","interval":"YEAR","price_cents":30000,"monthly_equivalent_cents":2500,"currency":"USD"}
],"status":"success"}
```

### 1.5 Subscribe (monthly) — `POST :8086/api/v1/subscription`   *(Bearer seller)*

```json
{"plan_code":"starter-monthly"}
```
**201**
```json
{"data":{"id":"1a2d65b9-8ec4-464b-b9e5-4d8e1dabec29","plan_id":"44049d78-a540-44bb-a656-a74baff9058a","status":"ACTIVE","current_period_start":"2026-09-10T10:54:22.432924Z","current_period_end":"2026-10-10T10:54:22.432924Z"},"status":"success"}
```

### 1.6 My subscription — `GET :8086/api/v1/subscription`

**200** — same object as 1.5.

### 1.7 Create the store (now allowed) — `POST :8081/api/v1/tenants`

```json
{"name":"Sam Electronics","slug":"sam-electronics"}
```
**201**
```json
{"data":{"id":"aa1fad3e-9317-468e-9689-a3bb5c2f7472","name":"Sam Electronics","slug":"sam-electronics","status":"ACTIVE","owner_user_id":"df883b51-0eb7-4a70-94af-b68a9e1c53d0","created_at":"2026-09-10T10:54:23.117919Z","updated_at":"2026-09-10T10:54:23.117919Z"},"status":"success"}
```

### 1.8 My memberships — `GET :8081/api/v1/me/memberships`

**200** — the creator is auto-added as **ADMIN**.
```json
{"data":[{"id":"24cb493c-f415-44e9-85bf-926f4746ab2c","tenant_id":"aa1fad3e-9317-468e-9689-a3bb5c2f7472","user_id":"df883b51-0eb7-4a70-94af-b68a9e1c53d0","role_id":"d9b0d46e-dee0-4943-a1f2-4da0cecfae6a","role_name":"ADMIN","created_at":"2026-09-10T10:54:23.117919Z","updated_at":"2026-09-10T10:54:23.117919Z"}],"status":"success"}
```

---

## Phase 2 — Building the catalog *(Bearer seller, all writes need ADMIN/MANAGER)*

### 2.1 Create category "Mobiles" — `POST :8082/api/v1/tenants/{T}/categories`

```json
{"name":"Mobiles","slug":"mobiles"}
```
**201**
```json
{"data":{"id":"b534729e-cca1-4fe7-9edf-24e13840a554","tenant_id":"aa1fad3e-…","name":"Mobiles","slug":"mobiles","position":0,"created_at":"2026-09-10T10:54:24.853251Z","updated_at":"2026-09-10T10:54:24.853251Z"},"status":"success"}
```
(A second category "Clothing" was created the same way → **201**.)

### 2.2 Add a VARIANT attribute "RAM" — `POST :8082/api/v1/tenants/{T}/categories/{C}/attributes`

```json
{"name":"RAM","code":"ram","role":"VARIANT"}
```
**201**
```json
{"data":{"id":"eedae797-3c40-4a9f-9f2f-390713a2889c","category_id":"b534729e-…","name":"RAM","code":"ram","role":"VARIANT","position":0,"values":[],"created_at":"2026-09-10T10:54:25.895127Z","updated_at":"2026-09-10T10:54:25.895127Z"},"status":"success"}
```

### 2.3 Add RAM values — `POST …/attributes/{A}/values`

```json
{"value":"8GB"}     → 201  {"data":{"id":"3fc815ab-…","value":"8GB","position":0},"status":"success"}
{"value":"12GB"}    → 201  {"data":{"id":"c56b8f9e-…","value":"12GB","position":0},"status":"success"}
```

### 2.4 Add a SPEC attribute "Chipset" — same endpoint as 2.2

```json
{"name":"Chipset","code":"chipset","role":"SPEC"}
```
**201** — `"role":"SPEC","values":[]`.

### 2.5 Bad attribute role — `POST …/attributes`

```json
{"name":"Bad","code":"bad","role":"WRONG"}
```
**422**
```json
{"status":"error","error":{"service":"product-service","code":"VALIDATION_ERROR","message":"one or more fields are invalid","method":"POST","path":"…/attributes","request_id":"cd46fc28-…","timestamp":"2026-09-10T10:54:28Z","fields":[{"field":"role","message":"must be one of: VARIANT, SPEC"}]}}
```

### 2.6 Create product (DRAFT) — `POST :8082/api/v1/tenants/{T}/products`

```json
{"name":"Galaxy S99","slug":"galaxy-s99","description":"Flagship phone"}
```
**201**
```json
{"data":{"id":"667f9530-f12a-4e84-b391-d3a2f56180cd","tenant_id":"aa1fad3e-…","name":"Galaxy S99","slug":"galaxy-s99","description":"Flagship phone","status":"DRAFT","created_at":"…","updated_at":"…","variants":[],"images":[],"category_ids":[],"options":[],"specs":[]},"status":"success"}
```

### 2.7 Attach product to a category — `PUT :8082/api/v1/tenants/{T}/products/{P}/categories`

```json
{"category_ids":["b534729e-cca1-4fe7-9edf-24e13840a554"]}
```
**200**  `{"data":{"category_ids":["b534729e-…"]},"status":"success"}`

### 2.8 Set display specs — `PUT :8082/api/v1/tenants/{T}/products/{P}/specs`

```json
{"specs":[{"attribute_id":"ae064663-6d9d-4c8b-9f9d-ec69bf7e6e6b","value":"Snapdragon 8 Gen 4"}]}
```
**200** — full product detail; note it now inherits `options` (RAM axis) from the
category and carries the `specs`:
```json
{"data":{ … "options":[{"code":"ram","name":"RAM","values":[{"id":"c56b8f9e-…","value":"12GB"},{"id":"3fc815ab-…","value":"8GB"}]}],
          "specs":[{"code":"chipset","name":"Chipset","value":"Snapdragon 8 Gen 4"}]},"status":"success"}
```

### 2.9 Create variants tied to a RAM value — `POST :8082/api/v1/tenants/{T}/products/{P}/variants`

```json
{"sku":"S99-8","price_cents":89900,"currency":"USD","is_default":true,
 "options":[{"attribute_id":"eedae797-…","value_id":"3fc815ab-…"}]}
```
**201**  `{"data":{"id":"155b8928-…","sku":"S99-8","price_cents":89900,"currency":"USD","is_default":true,…},"status":"success"}`

```json
{"sku":"S99-12","price_cents":99900,"currency":"USD",
 "options":[{"attribute_id":"eedae797-…","value_id":"c56b8f9e-…"}]}
```
**201** — `"sku":"S99-12","price_cents":99900` (12 GB variant is $10 dearer).

### 2.10 Variant option referencing an attribute not on the product — `POST …/variants`

```json
{"sku":"S99-X","price_cents":10000,"options":[{"attribute_id":"<a random uuid>","value_id":"3fc815ab-…"}]}
```
**422**
```json
{"status":"error","error":{"service":"product-service","code":"VALIDATION_ERROR","message":"that attribute is not defined for this product's categories, or has the wrong role","method":"POST","path":"…/variants","request_id":"0d02af12-…","timestamp":"2026-09-10T10:54:31Z"}}
```

---

## Phase 3 — Stock & checkout configuration *(Bearer seller)*

### 3.1 Register a SKU — `POST :8083/api/v1/tenants/{T}/inventory`

```json
{"sku":"S99-8","reorder_level":5}
```
**201**
```json
{"data":{"id":"7a9685b8-…","tenant_id":"aa1fad3e-…","sku":"S99-8","on_hand":0,"reserved":0,"available":0,"reorder_level":5,"low_stock":true,"created_at":"…","updated_at":"…"},"status":"success"}
```
(`S99-12` registered the same way → **201**.)

### 3.2 Receive stock — `POST :8083/api/v1/tenants/{T}/inventory/{SKU}/receive`

```json
{"quantity":50}      → 200  S99-8:  on_hand 50, available 50, low_stock false
{"quantity":30}      → 200  S99-12: on_hand 30, available 30
```

### 3.3 List inventory — `GET :8083/api/v1/tenants/{T}/inventory` → **200**

```json
{"data":{"items":[{"sku":"S99-12","on_hand":30,"reserved":0,"available":30,…},
                  {"sku":"S99-8","on_hand":50,"reserved":0,"available":50,…}],
         "limit":20,"offset":0},"status":"success"}
```

### 3.4 Tax config — `PUT :8085/api/v1/tenants/{T}/tax-config`

```json
{"tax_rate_bps":800,"tax_inclusive":false}
```
**200**  `{"data":{"tenant_id":"aa1fad3e-…","tax_rate_bps":800,"tax_inclusive":false},"status":"success"}`

### 3.5 Shipping rate — `POST :8085/api/v1/tenants/{T}/shipping-rates`

```json
{"name":"Standard","amount_cents":500,"free_over_cents":20000}
```
**201**  `{"data":{"id":"f7adef82-…","name":"Standard","amount_cents":500,"free_over_cents":20000,"active":true},"status":"success"}`

### 3.6 Coupon — `POST :8085/api/v1/tenants/{T}/coupons`

```json
{"code":"SAVE10","kind":"PERCENT","value":10,"min_order_cents":5000}
```
**201**  `{"data":{"id":"5ed61298-…","code":"SAVE10","kind":"PERCENT","value":10,"min_order_cents":5000,"redeemed_count":0,"active":true},"status":"success"}`

### 3.7 Coupon preview — `POST :8085/api/v1/tenants/{T}/coupon-check`

```json
{"code":"SAVE10","subtotal_cents":99900}
```
**200**  `{"data":{"valid":true,"discount_cents":9990},"status":"success"}`

### 3.8 Publish the product — `PATCH :8082/api/v1/tenants/{T}/products/{P}`

```json
{"status":"ACTIVE"}
```
**200** — full detail; each variant now carries its resolved `options`:
```json
"variants":[
  {"sku":"S99-8","price_cents":89900,"is_default":true,"options":{"ram":"8GB"},…},
  {"sku":"S99-12","price_cents":99900,"is_default":false,"options":{"ram":"12GB"},…}
]
```

---

## Phase 4 — Customer buys

### 4.1 Register customer — `POST :8081/api/v1/auth/register`

```json
{"name":"Cara Customer","email":"cust_1789037677@buyer.test","password":"cust@pass1","phone":"01720000002"}
```
**201** — `{access_token, refresh_token, expires_in:900}`. `auth/me` → id `606aaa1b-6417-42b3-87da-a678d081385b`.

### 4.2 Browse the public catalog — `GET :8082/api/v1/tenants/{T}/catalog` *(no auth)*

**200**
```json
{"data":{"items":[{"id":"667f9530-…","name":"Galaxy S99","slug":"galaxy-s99","description":"Flagship phone","status":"ACTIVE",…}],"limit":20,"offset":0},"status":"success"}
```

### 4.3 Full-text search — `GET :8082/api/v1/tenants/{T}/catalog?q=galaxy`

**200** — returns the same product (matched on name via `websearch_to_tsquery`).

### 4.4 Product detail — `GET :8082/api/v1/tenants/{T}/catalog/{P}` *(no auth)*

**200** — `variants[]` (with `options:{"ram":…}`), `options[]` (axes to render
selectors), `specs[]` (display lines). Picking `ram=12GB` → variant `S99-12` →
`price_cents 99900`.

### 4.5 Add to cart — `POST :8084/api/v1/tenants/{T}/cart/items` *(Bearer customer)*

```json
{"product_id":"667f9530-…","sku":"S99-12","quantity":2}
```
**201**
```json
{"data":{"id":"f5441265-…","customer_id":"606aaa1b-…","status":"ACTIVE","currency":"USD",
  "items":[{"id":"fb515d73-…","sku":"S99-12","quantity":2,"unit_price_cents":99900,"product_name":"Galaxy S99","subtotal_cents":199800}],
  "item_count":2,"subtotal_cents":199800,…},"status":"success"}
```
Prices are **snapshotted** from product-service at add time.

### 4.6 View cart — `GET :8084/api/v1/tenants/{T}/cart` → **200** (same as above).

### 4.7 Checkout — `POST :8085/api/v1/tenants/{T}/orders` *(Bearer customer, header `Idempotency-Key: idem-…`)*

```json
{"shipping_address":{"recipient_name":"Cara Customer","phone":"01720000002","address_line_1":"12 Test Rd","city":"Dhaka","postal_code":"1207","country":"BD"},
 "shipping_rate_id":"f7adef82-7288-4d3e-bcc9-64c8d4bbc0bb",
 "coupon_code":"SAVE10"}
```
**201**
```json
{"data":{"id":"dc2ca7d7-…","status":"PENDING_PAYMENT","currency":"USD",
  "subtotal_cents":199800,"discount_cents":19980,"shipping_cents":0,"tax_cents":14386,"grand_total_cents":194206,
  "item_count":2,"coupon_code":"SAVE10",
  "shipping_address":{…},"billing_address":{…same…},"payment_status":"PENDING",
  "items":[{"id":"6154763c-…","sku":"S99-12","quantity":2,"unit_price_cents":99900,"subtotal_cents":199800}],
  "created_at":"…","updated_at":"…"},"status":"success"}
```
Money math: subtotal 199800 − 10% (19980) = 179820; shipping **free** (over
$200 threshold); tax 8% of 179820 = 14386; **grand total 194206**.

### 4.8 Retry checkout, same `Idempotency-Key` — same request

**200** (not 201) and the **same order id** `dc2ca7d7-…` — no duplicate order.

### 4.9 Order detail — `GET :8085/api/v1/tenants/{T}/orders/{O}` → **200** (same object).

### 4.10 Pay — cash on delivery — `POST :8085/api/v1/tenants/{T}/orders/{O}/pay`

```json
{"payment_method":"COD"}
```
**200**
```json
{"data":{"id":"dc2ca7d7-…","status":"CONFIRMED","payment_method":"COD","payment_status":"PENDING","payment_id":"e59edb65-…", … "grand_total_cents":194206,…},"status":"success"}
```
COD → order `CONFIRMED` immediately, `payment_status` stays `PENDING` (cash
collected later).

### 4.11 Notifications (Kafka-driven) — `GET :8088/api/v1/notifications` *(Bearer customer)*

**200** — two notifications, newest first:
```json
{"data":{"items":[
  {"id":"bcb11814-…","type":"order.confirmed","title":"Order confirmed","body":"Your order is confirmed and will be prepared for shipment.","data":{"order_id":"dc2ca7d7-…","grand_total_cents":194206,…},"read":false,"created_at":"2026-09-10T10:54:43.850051Z"},
  {"id":"c33f65ea-…","type":"order.placed","title":"Order placed","body":"We've received your order and are holding your items.","data":{…},"read":false,"created_at":"2026-09-10T10:54:41.513839Z"}
],"limit":20,"offset":0},"status":"success"}
```
`GET /api/v1/notifications/unread-count` → `{"data":{"unread":2},"status":"success"}`.

### 4.12 Emails (Kafka-driven) — Mailpit `http://localhost:8025`

mail-service consumed the same events and produced:

| To | Subject |
|---|---|
| cust_…@buyer.test | `Order dc2ca7d7 received` |
| cust_…@buyer.test | `Receipt for order dc2ca7d7` |
| cust_…@buyer.test | `Order dc2ca7d7 is on its way` |

---

## Phase 5 — Fulfilment, return, review

### 5.1 Fulfil with tracking — `POST :8085/api/v1/tenants/{T}/orders/{O}/fulfil` *(Bearer seller)*

```json
{"carrier":"DHL","tracking_number":"DHL123456789"}
```
**200**
```json
{"data":{"id":"dc2ca7d7-…","status":"FULFILLED","payment_status":"PAID","tracking_carrier":"DHL","tracking_number":"DHL123456789","fulfilled_at":"2026-09-10T10:54:48.848982Z","paid_at":"2026-09-10T10:54:50.213811Z",…},"status":"success"}
```

### 5.2 Inventory after fulfil — `GET :8083/api/v1/tenants/{T}/inventory/S99-12` → **200**

`on_hand` **30 → 28** (shipped 2). `reserved 0`.

### 5.3 Request a return — `POST :8085/api/v1/tenants/{T}/orders/{O}/returns` *(Bearer customer)*

```json
{"reason":"Changed my mind","items":[{"sku":"S99-12","quantity":1}]}
```
**201**
```json
{"data":{"id":"8bc7703c-…","order_id":"dc2ca7d7-…","status":"REQUESTED","reason":"Changed my mind","refund_cents":99900,"restocked":false,
  "items":[{"sku":"S99-12","quantity":1,"unit_price_cents":99900,"product_name":"Galaxy S99","line_value_cents":99900}],"created_at":"…"},"status":"success"}
```

### 5.4 Seller lists returns — `GET :8085/api/v1/tenants/{T}/returns` → **200** (the item above).

### 5.5 Resolve the return — `POST :8085/api/v1/tenants/{T}/returns/{R}/resolve` *(Bearer seller)*

```json
{"approve":true}
```
**200**
```json
{"data":{"id":"8bc7703c-…","status":"COMPLETED","refund_cents":99900,"restocked":true,"resolved_at":"2026-09-10T10:54:54.194314Z",…},"status":"success"}
```
Refund goes through payment-service; stock is restocked via inventory-service.

### 5.6 Inventory after restock — `GET :8083/…/inventory/S99-12` → **200**

`on_hand` **28 → 29** (1 unit back).

### 5.7 Customer review — `POST :8082/api/v1/tenants/{T}/catalog/{P}/reviews` *(Bearer customer)*

```json
{"rating":5,"title":"Great","body":"Fast phone"}
```
**201**  `{"data":{"id":"ac31bbf0-…","rating":5,"title":"Great","body":"Fast phone",…},"status":"success"}`

### 5.8 Public reviews — `GET :8082/api/v1/tenants/{T}/catalog/{P}/reviews` *(no auth)*

**200**
```json
{"data":{"items":[{"id":"ac31bbf0-…","rating":5,"title":"Great","body":"Fast phone",…}],
  "limit":20,"offset":0,"summary":{"average":5,"count":1}},"status":"success"}
```

---

## Negative / error cases

| # | Call | Result |
|---|---|---|
| a | `GET :8081/api/v1/auth/me` no token | **401** `UNAUTHORIZED` — "authentication required" |
| b | `POST :8081/api/v1/auth/login` wrong password | **401** `UNAUTHORIZED` — "invalid credentials" |
| c | Customer `POST :8082/.../categories` (not a member) | **403** `FORBIDDEN` — "you are not a member of this tenant" |
| d | Duplicate category slug `mobiles` | **409** `CONFLICT` — "a category with that slug already exists" |
| e | Checkout with an empty cart | **422** `VALIDATION_ERROR` — "cart is empty" |
| f | `POST :8081/api/v1/auth/login` body `{bad json` | **400** `BAD_REQUEST` — "request body is not valid JSON" |
| g | `GET :8085/.../orders/00000000-0000-0000-0000-000000000000` | **404** `NOT_FOUND` — "order not found" |

Full bodies:

```json
// a
{"status":"error","error":{"service":"user-management","code":"UNAUTHORIZED","message":"authentication required","method":"GET","path":"/api/v1/auth/me","request_id":"43d1ebf9-…","timestamp":"2026-09-10T10:54:55Z"}}
// b
{"status":"error","error":{"service":"user-management","code":"UNAUTHORIZED","message":"invalid credentials","method":"POST","path":"/api/v1/auth/login","request_id":"9adfd0ce-…","timestamp":"2026-09-10T10:54:56Z"}}
// c
{"status":"error","error":{"service":"product-service","code":"FORBIDDEN","message":"you are not a member of this tenant","method":"POST","path":"/api/v1/tenants/aa1fad3e-…/categories","request_id":"9357d50c-…","timestamp":"2026-09-10T10:54:56Z"}}
// d
{"status":"error","error":{"service":"product-service","code":"CONFLICT","message":"a category with that slug already exists","method":"POST","path":"…/categories","request_id":"cbcc66b8-…","timestamp":"2026-09-10T10:54:56Z"}}
// e
{"status":"error","error":{"service":"order-service","code":"VALIDATION_ERROR","message":"cart is empty","method":"POST","path":"…/orders","request_id":"1fb99b0f-…","timestamp":"2026-09-10T10:54:57Z"}}
// f
{"status":"error","error":{"service":"user-management","code":"BAD_REQUEST","message":"request body is not valid JSON","method":"POST","path":"/api/v1/auth/login","request_id":"e4263ea7-…","timestamp":"2026-09-10T10:54:57Z"}}
// g
{"status":"error","error":{"service":"order-service","code":"NOT_FOUND","message":"order not found","method":"GET","path":"…/orders/00000000-0000-0000-0000-000000000000","request_id":"a4150744-…","timestamp":"2026-09-10T10:54:57Z"}}
```

---

## Problems found & fixed during this test

| # | Problem | Fix |
|---|---|---|
| 1 | **Checkout create response returned order items with a zero UUID** (`"id":"00000000-0000-0000-0000-000000000000"`). The multi-row `INSERT INTO order_items` doesn't `RETURNING` ids, and the handler returned the pre-insert in-memory slice. The idempotent re-fetch and `GET /orders/{id}` were already correct. | `order-service/internal/order/services/service.go` — after `repo.Create`, reload items with `repo.ListItems(order.ID)` before building the response. |
| 2 | **mail-service / payment-service running in stub mode** — their containers still had the old env (`MAIL_SMTP_HOST` empty, `PAYMENT_STRIPE_SECRET_KEY` empty) because they were started before those values were added to `.env.dev`. | Recreated both containers. mail-service log now `smtp relay: mailpit`; payment-service now returns a real Stripe PaymentIntent (`pi_3UE5sR…` not `pi_stub_…`) and leaves the order `PENDING_PAYMENT` until the card is confirmed. |
| 3 | **mail-service could not send even to Mailpit** — `net/smtp` was handed `smtp.PlainAuth` with empty credentials, so it aborted with `smtp: server doesn't support AUTH`. | `mail-service/internal/smtp/smtp.go` — pass a `nil` auth when no username is configured. |
| 4 | **After #3, Mailpit rejected the envelope** — `501 5.5.4 invalid FROM parameter`. `MAIL_SMTP_FROM` (`Platform (dev) <no-reply@localhost>`) was passed as the SMTP `MAIL FROM` command, which must be a bare address. | `mail-service/internal/smtp/smtp.go` — keep the display-name string for the `From:` header, but parse out the bare address (`net/mail.ParseAddress`) for the envelope sender. |

### Second pass — after the fixes

| Check | Result |
|---|---|
| Checkout response item ids | real UUIDs (`5544fb80-…`) |
| CARD pay (`{"payment_method":"CARD"}`) | **200**, real `client_secret` `pi_3UE5sRBPQJ9jcRLH0sM7E3KZ_secret_…`, order stays `PENDING_PAYMENT`, `payment_status:"PENDING"` (awaits Stripe.js confirmation / webhook) |
| Emails in Mailpit after a COD order + fulfil | 3 messages: `Order … received`, `Receipt for order …`, `Order … is on its way` |

Everything else in the flow — subscription gate, tenant roles, attribute/variant/spec model, price snapshotting, idempotent checkout, money math (discount → free-shipping threshold → tax), COD confirmation, Kafka notifications, tracking, partial refund + restock on return, reviews, and all seven error cases — behaved correctly on the first run.

To make the two stub→live fixes stick, `mail-service` and `payment-service` must be
recreated (not just restarted) so they pick up `.env.dev`:
```
docker compose --env-file .env.dev -f deployments/compose/docker-compose.dev.yml up -d mail-service payment-service
```

---

## Gap-analysis fixes — implementation + real API verification (2026-09-11)

Full write-up of the gaps: `ApiTestResults/GAP_ANALYSIS.md`. This section implements
and **exercises through the real Dockerized services** the P0 and P1 items from
that report. No new `*_test.go` files — every check below is a real HTTP call
(or a validly-signed simulated Stripe webhook, since there's no way to drive a
real card confirmation headlessly) against the running containers, with the
actual response and the actual database/Kafka-driven side effect captured.

Webhook simulation method: payment-service verifies `Stripe-Signature` with
`HMAC-SHA256(webhook_secret, "{timestamp}.{payload}")` (`internal/gateway/gateway.go`).
A small script signs a payload with the real `PAYMENT_STRIPE_WEBHOOK_SECRET` the
same way and POSTs it to `POST /api/v1/webhooks/stripe` — this exercises the
actual signature-verification and event-processing code, it just stands in for
Stripe's own delivery (the PaymentIntent itself is real — `payment-service`
calls the real Stripe test-mode API to create it during checkout).

### P0.1 — Stripe webhook → order synchronization

**Root cause:** `payment-service` published `payment.captured` to Kafka, but
`order-service` had no consumer for it (the `events.Consumer` type existed,
unused). An order only ever moved off `PENDING_PAYMENT` when a client called
`POST /orders/:id/pay` again.

**Fix:** `order-service` gained a `payment-events` Kafka consumer
(`internal/order/eventhandler.go`, wired in `internal/server/server.go`) that
calls two new idempotent service methods, `ApplyPaymentCaptured` /
`ApplyPaymentFailed` (`internal/order/services/service.go`), mirroring the
existing mail-service/notification-service consumer pattern.

**Files changed:** `order-service/internal/events/events.go` (+`PaymentFailed`
const), `internal/order/eventhandler.go` (new), `internal/order/services/service.go`,
`internal/order/domain/order.go`, `internal/order/repository/repository.go`,
`internal/order/routes/routes.go` (now returns the wired `*services.Service`),
`internal/server/{server.go,shutdown.go,wire.go}`. Same `payment.failed` const
added to payment-service/mail-service/notification-service's copies of
`internal/events/events.go` for consistency (mail/notification already had
handling wired in this pass too — see P0.2).

**Test — checkout → pay → simulated `payment_intent.succeeded` webhook, no second `Pay()` call:**

Request (checkout, then `POST /orders/:id/pay {"payment_method":"CARD"}`, then the signed webhook):
```
POST /api/v1/webhooks/stripe
Stripe-Signature: t=<ts>,v1=<hmac>
{"type":"payment_intent.succeeded","data":{"object":{"id":"pi_3UERiJBPQJ9jcRLH2Knrw03L"}}}
```
Response: `200`, empty body.

Order **before** the webhook: `status=PENDING_PAYMENT payment_status=PENDING`.
Order **after** the webhook (no `Pay()` call in between): `status=CONFIRMED payment_status=PAID paid_at=2026-09-11T10:29:50.812577Z`.

Notifications created (Kafka-driven, via the new consumer):
```
order.confirmed  Order confirmed
payment.captured Payment received
order.placed     Order placed
```
Mailpit: `Order c4f1c7c1 received`, `Receipt for order c4f1c7c1`.

**PASS** — the order is fully synced from the webhook alone.

### P0.2 — Failed CARD payment releases the inventory reservation

**Root cause:** `Pay()` handled `payment.Status == "CAPTURED"` but had no branch
for `"FAILED"` — a declined/cancelled card left the checkout-time reservation
in place forever, with no automatic path back to available stock.

**Fix:** `releaseOnPaymentFailure` (`order/services/service.go`) — marks
`payment_status='FAILED'` via a new, guarded `MarkPaymentFailed` repo method
(no-ops once payment_status has moved off `PENDING`, so it can't double-fire),
then releases the reservation. Called from both the synchronous `Pay()` path
(`payment.Status == "FAILED"`) and the new async `ApplyPaymentFailed` Kafka path
(P0.1), so a decline is caught however it's discovered.

**Test — checkout (qty 2) → CARD → simulated `payment_intent.payment_failed` webhook:**

```
inventory BEFORE:               on_hand 29  reserved 0  available 29
checkout (qty 2) reserves:      on_hand 29  reserved 2  available 27
webhook payment_intent.payment_failed  -> HTTP 200
order after:                    status=PENDING_PAYMENT  payment_status=FAILED
inventory after (RELEASED):     on_hand 29  reserved 0  available 29   ✅ back to baseline
```

**Duplicate failure delivery** (same signed payload posted twice) — inventory
unchanged (`on_hand 29 reserved 0 available 29`), no double release.

**Client re-syncs by calling `Pay()` again on the now-FAILED order** — `Sync()`
short-circuits (`p.Status != Pending`), inventory still `reserved 0`, no
double-release, order's `payment_status` stays `FAILED`.

**PASS** — release happens exactly once regardless of how many times the
failure is observed.

### P0.3 — Payment webhook concurrency / idempotency

**Root cause:** `HandleWebhook` did a plain `SELECT` (no lock) then a plain
`UPDATE`. Two genuinely concurrent deliveries of the same event could both pass
the "still PENDING" guard before either committed, both capture, both publish
`payment.captured` — duplicate emails/notifications.

**Fix:** `HandleWebhook` (`payment-service/internal/payment/services/service.go`)
now runs the read-check-write inside one DB transaction with the payment row
locked (`GetByGatewayRefForUpdate` — new repo method, `SELECT ... FOR UPDATE`,
same pattern inventory-service already used for stock). The Kafka publish only
happens after the transaction commits, and only on the delivery that actually
changed something.

**Test — 8 truly concurrent identical webhook deliveries** (backgrounded curl,
`wait`) for one freshly-created order:

```bash
for i in $(seq 1 8); do curl ... -d "$PAYLOAD" -H "Stripe-Signature: $SIG" & done; wait
```
All 8 responses: `200`.

```
payments row:   status=CAPTURED  captured_at=2026-09-11 10:30:52.635758+00  refunded_cents=0   (ONE row, ONE timestamp)
notifications:  {'order.confirmed': 1, 'payment.captured': 1, 'order.placed': 1}                (exactly once each)
mailpit:        "Receipt for order db8abb0d", "Order db8abb0d received"                          (exactly 2 messages, not 16)
```

**PASS** — proven under real concurrent load, not just sequential retries.

### P1.4 — Coupon redemption consistency

**Root cause:** `RedeemCoupon` ran *after* the order was already committed with
the discount applied, and its error was only logged — a failed redemption
(e.g. lost the race against `max_redemptions`) left the order discounted with
the coupon's counter never incremented.

**Fix:** `order/repository.Repository.Create` now takes an optional `redeem
func(tx *sql.Tx) error` that runs **inside the same transaction** as the order
+ items insert (`internal/order/services/service.go` `Checkout`,
`internal/order/repository/repository.go`, `internal/order/domain/order.go`).
A failed redemption rolls the whole order back — no discount without a real,
committed redemption, and vice versa.

**Test — sequential exhaustion**, coupon `ONEUSE` (`max_redemptions=1`):

```
1st checkout with ONEUSE:  {"status":"success","data":{"discount_cents":500}}
coupon state after:        {"code":"ONEUSE","max_redemptions":1,"redeemed_count":1}

2nd checkout with ONEUSE:  {"status":"error","error":{"code":"VALIDATION_ERROR",
                             "message":"that coupon has reached its redemption limit"}}
coupon state after:        {"code":"ONEUSE","max_redemptions":1,"redeemed_count":1}   <- unchanged, no phantom order
```

**Test — 6-way concurrent redemption race**, fresh coupon `RACE1` (`max_redemptions=1`),
6 different customers checking out simultaneously with it:

```sql
select count(*) from orders where coupon_code='RACE1';   -- 1
select redeemed_count from coupons where code='RACE1';   -- 1
```

**PASS** — exactly one of the six concurrent attempts succeeded; the coupon's
counter and the actual order count agree under real concurrency, both capped at 1.

### P1.5 — Duplicate / over-return quantity prevention

**Root cause:** `Request()` validated a return's quantity only against the
*original* order line quantity, never against quantity already covered by a
REQUESTED or COMPLETED return on the same order — a SKU could be "returned"
past what was actually ordered.

**Fix:** new batched repo method `AlreadyReturnedBySKU` (one query, sums
non-rejected return items per SKU for the order) called once per `Request()`
(`internal/returns/repository/repository.go`,
`internal/returns/services/service.go`), plus a new domain error
`ErrReturnQuantityExceeded`. A REJECTED return's quantity is excluded from the
sum, so rejecting one frees its quantity back up.

**Test — order of 3× `S99-8`, fulfilled:**

```
1) return qty 1 (of 3)                    -> success -> manager approves -> COMPLETED, refund 89900
2) return qty 3 (only 2 left)             -> 422 VALIDATION_ERROR "that quantity has already been returned or is pending return"
3) return qty 2 (exactly what's left)     -> success (status REQUESTED, not yet resolved)
4) return qty 1 more (nothing left, even  -> 422 VALIDATION_ERROR (same message) -- pending REQUESTED already counts
   though #3 is only REQUESTED, not COMPLETED)
5) manager REJECTS return #3              -> REJECTED
6) return qty 2 again after the rejection -> success (rejected quantity correctly freed back up)
```

**PASS** — every one of the required edge cases (duplicate request, over-limit,
remaining-exact, pending-counts-too, rejection-frees-it-back-up) behaves correctly.

### P1.6 — Tenant-scoping hardening

**Root cause:** ~12 repository mutation methods across order/payment/product/user-management
filtered by `id` only, not `tenant_id + id` — the security audit found **no live
exploit** (every call site already re-verified tenant ownership in the service
layer first), but the SQL itself provided no independent defense.

**Fix:** added `tenant_id` (or `product_id`, for the tables that don't carry a
tenant column) to the WHERE clause of every flagged method: order-service's
`AttachPayment/MarkConfirmed/MarkPaymentPaid/MarkPaymentFailed/MarkPaymentRefunded/
MarkFulfilled/MarkCancelled` and `order_returns.Resolve`; payment-service's
`payments.Update`; product-service's `product_variants.GetByID/Update/Delete/
ClearDefault` and `product_images.GetByID/Update/Delete`; user-management's
`memberships.UpdateRole/Delete`. Since these were never independently
reachable, there is no black-box HTTP scenario that shows *new* rejection
behavior — the correct check was already there one layer up. What real API
tests can and do confirm: same-tenant operations still work (no regression from
adding the WHERE clause) and cross-tenant access is still blocked exactly as
before.

**Test:**
```
PATCH /tenants/{TENANT_A}/products/{P}/variants/{V}  (Bearer: Tenant A admin, own variant)
  -> 200 {"status":"success","data":{"barcode":"REGCHECK123", ...}}                  regression check: PASS

Tenant B created fresh (register -> subscribe -> create tenant) to confirm the
subscription-gate + auto-ADMIN-membership business rules still hold:
  -> 201, new tenant id, Tenant B's owner is auto-ADMIN

PATCH /tenants/{TENANT_A}/products/{P}/variants/{V}  (Bearer: Tenant B admin)
  -> 403 {"status":"error","error":{"code":"FORBIDDEN","message":"you are not a member of this tenant"}}

GET .../products/{P} as Tenant A admin afterward -> barcode still "REGCHECK123" (untouched by the blocked attempt)
```

**PASS** — no regression on legitimate same-tenant writes; cross-tenant access
remains correctly blocked; the subscription→tenant-creation→auto-ADMIN business
rule is intact.

### Build/runtime verification

All 8 services: `gofmt -l` clean, `go build ./...` clean, `go vet ./...` clean
(no test files added or changed). Recreated the 5 touched containers
(`order-service`, `payment-service`, `product-service`, `user-management`,
`notification-service`) and confirmed clean boot logs — `order-service
consuming payment-events` (new consumer active), zero panics/fatals across all
five services' logs for the full test session.

### Remaining from GAP_ANALYSIS.md (not implemented this pass)

Password reset / change-password flow (new feature, no existing scaffold to
extend), cross-service session revocation (architecture trade-off — needs your
call), subscription renewal / ongoing enforcement (needs your call on scope),
product hard-delete guard, review purchase-eligibility (needs a cross-service
design decision), per-user coupon limit, Kafka DLQ, `PARTIALLY_REFUNDED`
order-payment-status — all still open, see `GAP_ANALYSIS.md` §8 for the full
list and reasoning. `order-cancel-voids-pending-PaymentIntent` is now
implemented — see the next section.

---

## Order cancellation lifecycle — implementation + real API verification (2026-09-11)

Built on top of the payment webhook work above (webhook → order sync, the
concurrency-safe `HandleWebhook`, and idempotent `ApplyPaymentCaptured`/
`ApplyPaymentFailed`). Cancellation itself (`POST /orders/:id/cancel`) already
existed and worked — this pass closes the one real gap: **a delayed Stripe
capture landing after the order was already cancelled must never resurrect
it**, plus adds voiding the pending Stripe PaymentIntent at cancel time so
that race is rare in the first place, not just survivable.

### What was already there (inspected, not rebuilt)

- `POST /tenants/:tenantId/orders/:orderId/cancel` — `guards.Authenticated`,
  owner-or-manager enforced inside the service (`load()`).
- `MarkCancelled` (`order/repository/repository.go`) is DB-state-guarded:
  `WHERE status IN ('PENDING_PAYMENT','CONFIRMED')` — so FULFILLED and
  already-CANCELLED orders are rejected at the SQL layer, not by an
  if/else chain in Go. This is the state machine; nothing new was invented.
- Inventory release, refund-if-captured, and the `order.cancelled` Kafka event
  all already existed and were unchanged.

### What was missing and got implemented

**1. Void the pending Stripe PaymentIntent at cancel time.** `Cancel()`
(`order/services/service.go`) gained a second branch alongside the existing
"refund if already PAID" one: if the payment is CARD and still PENDING, it
calls a new payment-service endpoint to cancel the PaymentIntent — best
effort, same style as the existing refund call (log and continue, don't fail
the cancellation over it).

New payment-service capability (mirrors the existing `Refund`/`Settle` shape
exactly): `Gateway.Cancel` (`internal/gateway/gateway.go`, real Stripe →
`POST /payment_intents/:id/cancel`; stub → no-op), `Service.Cancel`
(`internal/payment/services/service.go`, only a `PENDING` payment can be
cancelled — `ErrNotCancellable` otherwise), handler + route
`POST /internal/tenants/:tenantId/payments/:paymentId/cancel`, and a matching
`paymentclient.Cancel` in order-service.

**2. `ApplyPaymentCaptured` must not resurrect a cancelled order.** Before this
pass it only handled `PENDING_PAYMENT`/`CONFIRMED`. It now switches on the
order's actual status: for `CANCELLED`, it never calls `advanceToConfirmed` —
instead it records the real capture (`MarkPaymentPaid`, best effort) and
immediately reverses it through the **existing** refund path
(`s.payments.Refund`, the same call `Cancel()` itself uses for an
already-paid order) — same mechanism as a manager refunding a paid order, not
a new workflow. `FULFILLED` is a no-op (already fully settled).

**3. `ApplyPaymentFailed`/`releaseOnPaymentFailure` must not double-release
inventory.** It now re-checks the order's status before releasing: only a
still-`PENDING_PAYMENT` order gets its stock released here. An already-
`CANCELLED` order's stock was released by `Cancel()` itself — releasing again
would double-release the same units back into `available`.

**Files changed:** `payment-service/internal/{gateway/gateway.go,
payment/{domain/{payment.go,errors.go},services/service.go,
handler/http/handler.go,routes/routes.go},httpx/errors.go}`;
`order-service/internal/{paymentclient/paymentclient.go,
order/services/service.go}`. No migrations needed (no new DB columns; the
payment reuses its existing `FAILED` status for "voided, never captured").

### Test 1 — COD order cancellation

```
inventory before checkout:  on_hand 47 reserved 8  available 39
checkout (qty 1):           on_hand 47 reserved 9  available 38
pay COD:                    order status=CONFIRMED payment_status=PENDING
POST .../cancel:            200 {"status":"CANCELLED", "payment_status":"PENDING", "cancelled_at":"..."}
inventory after cancel:     on_hand 47 reserved 8  available 39   <- back to baseline
duplicate POST .../cancel:  409 CONFLICT "the order cannot move to that state from its current one"
inventory after duplicate:  on_hand 47 reserved 8  available 39   <- unchanged, no double release
```
**PASS** — COD payment_status correctly stays PENDING (nothing was ever
collected), inventory releases exactly once, duplicate cancel is safely
rejected.

### Test 2 — CARD, still PENDING_PAYMENT, cancel voids the PaymentIntent (the common case)

```
checkout + pay CARD:        order PENDING_PAYMENT, payment PENDING, gateway_ref=pi_3UESAu...
inventory after reserve:    reserved 9  available 38
POST .../cancel:            200 {"status":"CANCELLED", "payment_status":"PENDING", ...}
payments row after cancel:  status=FAILED  (our Cancel() successfully voided the real Stripe test-mode PaymentIntent)
inventory after cancel:     reserved 8  available 39   <- released
```

**The race — delayed webhook after cancellation, must not resurrect:**
```
POST /api/v1/webhooks/stripe  {"type":"payment_intent.succeeded", ...same gateway_ref...}
  -> HTTP 200 (payment-service's own guard: status is already FAILED, not
     PENDING, so HandleWebhook no-ops it before it ever reaches order-service)
order after the delayed webhook:  status=CANCELLED  payment_status=PENDING   <- unchanged
inventory after the delayed webhook: reserved 8  available 39                 <- unchanged
duplicate delivery of the same delayed webhook: HTTP 200, order state stable
```
**PASS** — voiding at cancel time means the natural case never even reaches
the order-side race logic; the order simply never moves.

### Test 3 — the hard race: void itself gets rejected, capture lands anyway

Simulated by pointing the payment's `gateway_ref` at a nonexistent PaymentIntent
id right before cancelling, so payment-service's real Stripe void call 404s and
the payment is left `PENDING` — the state a genuine "Stripe already captured a
moment before the cancel" race would also leave it in.

```
cancel (void rejected by Stripe, logged, order still cancels): HTTP 200
state right after:            order status=CANCELLED  payment_status=PENDING (still)
                               payments row: status=PENDING refunded_cents=0
delayed payment_intent.succeeded webhook (same fake gateway_ref):  HTTP 200
order AFTER the late capture:  status=CANCELLED  payment_status=PAID   <- captured recorded, but NOT resurrected
```
The subsequent auto-refund call to real Stripe then 404s too — because
nothing in this sandbox ever drives a genuine Stripe card confirmation, *no*
PaymentIntent used anywhere in this test session is ever really captured at
Stripe, so a real refund call on one always fails the same way (confirmed in
payment-service's own log: `stripe POST /refunds: status 404: resource
missing`). That's a sandbox limitation, not a code defect — retried 3x by the
Kafka consumer, then logged and dropped, **and critically the order stayed
CANCELLED throughout**, which is the property that actually matters.

To confirm the refund *wiring* itself (not the Stripe network call) is
correct, the same race was repeated with the payment's `method` flipped to
`COD` in the DB right before the delayed webhook fires — `Refund()` only calls
Stripe for CARD, so this isolates the application logic from the Stripe call:
```
order after the late capture:  status=CANCELLED  payment_status=REFUNDED
payments row:                  status=REFUNDED  amount_cents=97092  refunded_cents=97092
notifications:                 payment.refunded - Payment refunded
                                payment.captured - Payment received
                                order.cancelled  - Order cancelled
```
**PASS** — order never resurrected in either variant; with a refundable
payment object, the full "capture recorded → auto-refunded → notified" chain
completes correctly and reuses the existing refund + notification pipeline.

### Test 4 — CARD order already CONFIRMED + PAID → cancel → refund (existing path, unchanged)

```
checkout + pay CARD + normal (non-race) capture webhook:  status=CONFIRMED payment_status=PAID
POST .../cancel:                                           200, status=CANCELLED
```
Refund is attempted via the same pre-existing `s.payments.Refund` call this
feature didn't change; it 404s against real Stripe for the same "nothing was
ever genuinely captured in this sandbox" reason as Test 3 — this is identical,
unmodified behavior from before this pass, not a regression.
**PASS** for the part in scope (cancel transitions correctly and attempts
the existing refund path; no code in this area was changed).

### Test 5 — invalid state: cancelling a FULFILLED order

```
checkout -> pay COD -> fulfil (DHL, tracking F1)
POST .../cancel:  409 CONFLICT {"message":"the order cannot move to that state from its current one"}
```
**PASS** — DB state guard (`WHERE status IN ('PENDING_PAYMENT','CONFIRMED')`)
rejects it, unchanged pre-existing behavior, confirmed still correct.

### Test 6 — authorization

```
non-owner customer cancels someone else's order:      403 FORBIDDEN "only a tenant ADMIN or MANAGER can do that"
a DIFFERENT tenant's admin cancels tenant A's order:   403 FORBIDDEN (same message — no membership in tenant A at all)
the actual owner cancels their own eligible order:     200
a tenant ADMIN cancels a customer's order in their own tenant: 200
```
**PASS** — ownership/manager boundary and tenant isolation both hold; the
legitimate owner and seller-initiated paths aren't over-blocked.

### Build/runtime verification

`payment-service` + `order-service`: `gofmt -l` / `go build ./...` / `go vet
./...` clean (no test files). Both containers force-recreated; confirmed clean
boot (`order-service consuming payment-events`), zero panics/fatals across the
full cancellation test session.

## Subscription lifecycle — implementation + real API verification (2026-09-11)

### What already existed (unchanged)

Subscription billing (module `billing`, table `subscriptions` + `plans`, all in
payment-service) was already real, not a stub in the "fake API" sense — `GET
/plans`, `GET/POST/DELETE /subscription`, and the internal `GET
/internal/users/:userId/subscription` entitlement check user-management's
tenant-creation gate calls were all wired and working. What was explicitly
**stubbed by design** (per the file's own header comment) is the *charge*:
`Subscribe` never touches a payment gateway — it just records `current_period
= [now, now+interval)` with `status = ACTIVE`. There is no
`stripe_subscription_id`/`stripe_customer_id` column and the `Gateway`
interface has no recurring-billing method — this is a manually-tracked "paid
until X" model, not a real Stripe `Subscription` object. The tenant-creation
gate itself (`user-management`'s `tenant/services.Create` →
`SubscriptionGuard.RequireActiveSubscription` → payment-service's internal
status route) was untouched and is preserved exactly as-is.

States already declared: `ACTIVE`, `CANCELED`, `EXPIRED`, `PAST_DUE`
(`billing/domain/billing.go`). Before this pass, only `ACTIVE`/`CANCELED` were
ever assigned — `EXPIRED` and `PAST_DUE` were dead enum values; a lapsed
subscription's `status` column stayed `ACTIVE` forever (entitlement was only
ever evaluated *incidentally*, via `IsEntitled`'s `now.Before(period_end)`
check), and — critically — this meant a lapsed subscription **could never be
renewed**: `Subscribe` always `INSERT`ed a new row, and the DB's own
`subscriptions_one_active_per_user` partial unique index (`WHERE status =
'ACTIVE'`) rejected it because the old, lapsed-but-still-`ACTIVE` row was
still occupying that slot. This was the actual root-cause bug closed by this
pass — not a missing feature so much as an existing one-way door.

### What was implemented

1. **Lazy expiry** (no cron/worker exists anywhere in this system, and adding
   one would be new infrastructure beyond scope) — every read of a user's
   subscription now first runs a scoped, indexed `UPDATE subscriptions SET
   status='EXPIRED' WHERE user_id=$1 AND status='ACTIVE' AND
   current_period_end <= now()` before selecting. This makes the stored
   `status` column agree with what `IsEntitled` already computed
   incidentally, and — by flipping the row out of `ACTIVE` — frees the
   partial unique index so the user can renew.
2. **Renewal / reactivation, unified into the existing `POST /subscription`**
   (no new billing architecture invented): the same endpoint now branches on
   the caller's most recent subscription row (locked `FOR UPDATE` inside a
   transaction):
   - no subscription ever existed → create a fresh `ACTIVE` row (unchanged
     behavior)
   - `ACTIVE` and not yet lapsed → **renewal**: extend `current_period_end`
     by one more interval *from the current end* (no paid-for time lost),
     same row
   - `CANCELED` / `EXPIRED` / `PAST_DUE` → **reactivate** that same row:
     `ACTIVE` again, a fresh period starting now
3. **Renewal idempotency**: an optional `Idempotency-Key` header (same
   convention as order checkout's) is stored on the row as
   `last_renewal_key`. A retried/duplicate renewal request carrying the same
   key is a no-op (returns the already-renewed row unchanged) instead of
   extending the period a second time. The row lock means two concurrent
   requests with the same key serialize instead of racing.
4. **Payment-failure grace state**: `POST
   /internal/users/:userId/subscription/mark-past-due` (X-Internal-Key) flips
   an `ACTIVE` subscription to `PAST_DUE` without touching
   `current_period_end` or anything tenant-side. This is the entry point a
   real Stripe `invoice.payment_failed` webhook would call once real
   recurring billing exists — today it's invoked directly because
   `Subscribe`/renew never performs a real charge that could fail on its own
   (see "Remaining gaps" below).
5. Two small schema/behavior additions: `subscriptions.last_renewal_key TEXT`
   column (migration `000005_billing_renewal`); `SubscriptionRepository.Update`
   now also persists `plan_id` (needed so renew-with-a-different-plan and
   reactivate-with-a-plan both work).

**Business-rule decision made explicit** (§6 of the task): this codebase
already keeps subscription *entitlement* (`subscriptions.user_id`) and tenant
*ownership/data* (`tenants.owner_id`, memberships, orders, products,
inventory) as separate concepts, coupled only at the one-time
tenant-creation gate — nothing else in the system re-checks the owner's
subscription. That is preserved exactly: an existing tenant's data and
day-to-day operations are **never** gated by its owner's subscription status,
regardless of ACTIVE/PAST_DUE/EXPIRED/CANCELED. The *only* enforcement is,
and remains, "no active subscription → can't create a (new) tenant." This
was a deliberate choice not to add new enforcement — building a
per-operation subscription check for existing tenants would be new
architectural coupling this codebase doesn't have today, and the task
explicitly said not to blindly block every API or destroy/restrict a working
store over a personal billing lapse.

### Files changed

- `payment-service/migrations/000005_billing_renewal.{up,down}.sql` (new)
- `payment-service/internal/billing/domain/billing.go` — `Subscription.LastRenewalKey`, `GetLatestByUserForUpdate` on the repository interface
- `payment-service/internal/billing/repository/repository.go` — `expireLapsed`, `GetLatestByUserForUpdate`, `Update` now also sets `plan_id`
- `payment-service/internal/billing/services/service.go` — `Service` gained a `db *sql.DB` field; `Subscribe` rewritten (transaction + row lock, renewal/reactivation/idempotency-key branching); new `MarkPastDue`
- `payment-service/internal/billing/handler/http/handler.go` — `Subscribe` reads the `Idempotency-Key` header; new `InternalMarkPastDue`
- `payment-service/internal/billing/routes/routes.go` — new internal route `POST /internal/users/:userId/subscription/mark-past-due`

### API changes

- `POST /api/v1/subscription` — same request/response shape; now also renews
  or reactivates instead of only ever creating. Accepts an optional
  `Idempotency-Key` header.
- New: `POST /api/v1/internal/users/:userId/subscription/mark-past-due`
  (X-Internal-Key) — payment-failure simulation, models what a real Stripe
  `invoice.payment_failed` webhook would trigger.

### Database changes

`subscriptions.last_renewal_key TEXT` (nullable), migration `000005`, applied
against the running `payment_db`.

### Webhook/event changes

None. Audited the existing Stripe webhook handler
(`payment/services/service.go HandleWebhook`) — it only recognizes
`payment_intent.succeeded` / `payment_intent.payment_failed` /
`payment_intent.canceled`, all order-payment events; there is no
`invoice.*`/`customer.subscription.*` handling because there is no real
Stripe `Subscription` object for those events to describe. Per the task's own
instruction ("do not invent a new webhook system... follow the existing
manual-periods model"), no subscription-specific Stripe webhook handling was
added. What *was* verified for real is that the existing webhook endpoint is
safe against subscription-shaped events it doesn't understand — see Test 7
below.

### Real API scenarios executed (all against the live Docker stack)

#### Test 1 — no subscription → tenant creation blocked

```
GET /api/v1/subscription (new user, never subscribed):  404 NOT_FOUND "no active subscription"
POST /api/v1/tenants:                                     422 VALIDATION_ERROR
  "an active subscription is required to create a store"
```
**PASS**

#### Test 2 — subscribe → ACTIVE → tenant creation succeeds → creator is ADMIN

```
POST /api/v1/subscription {"plan_code":"starter-monthly"}:
  201 {status: ACTIVE, current_period_start: 2026-09-11T11:34:20Z, current_period_end: 2026-10-11T11:34:20Z}
POST /api/v1/tenants {"name":"First Shop", ...}:  201 {id, owner_user_id, status: ACTIVE}
GET /api/v1/tenants/:id/members:  [{ user_id: <owner>, role_name: "ADMIN" }]
```
**PASS** — the pre-existing "subscribe → create tenant → creator = ADMIN"
chain is completely unaffected by this pass.

#### Test 3 — renewal extends the period; duplicate renewal does not double-extend

```
POST /subscription (same plan, Idempotency-Key: renew-key-X):
  current_period_end: 2026-10-11T11:34:20Z -> 2026-11-11T11:34:20Z   (+1 month from the OLD end, not from "now")
POST /subscription again, SAME Idempotency-Key:
  current_period_end: 2026-11-11T11:34:20Z (unchanged -- no-op, same row returned)
```
**PASS** — renewal extends without losing paid time; the duplicate request
(simulating a retried/duplicate renewal call) produced zero additional
extension.

#### Test 4 — renewal payment failure → PAST_DUE (grace state)

```
POST /internal/users/:userId/subscription/mark-past-due (X-Internal-Key):
  200 {status: PAST_DUE, current_period_end: unchanged}
GET /internal/users/:userId/subscription (the tenant-creation gate's own check):
  200 {"active": false}
POST /api/v1/tenants (new tenant, while PAST_DUE):
  422 VALIDATION_ERROR "an active subscription is required to create a store"
GET /api/v1/tenants/:TENANT1 (the EXISTING tenant created in Test 2):
  200 -- unchanged, fully operational, no data touched
GET /api/v1/tenants/:TENANT1/members:  200 -- membership untouched
```
**PASS** — new tenant creation correctly blocked; the already-existing tenant
is completely unaffected (no destruction, no restriction), matching the
business-rule decision above.

#### Test 5 — renewing from PAST_DUE restores ACTIVE and unblocks tenant creation

```
POST /subscription {"plan_code":"starter-monthly"}:
  201 {status: ACTIVE, current_period_start: <now>, current_period_end: <now+1mo>}
POST /api/v1/tenants ("Second Shop"):  201 -- succeeds again
```
**PASS**

#### Test 6 — expiration (period end reached) → EXPIRED, gated, data preserved, renewable

Real elapsed time can't be waited out in this sandbox, so `current_period_end`
was moved into the past directly in Postgres for this one subscription row
(the same fault-injection technique used throughout the cancellation-lifecycle
pass) to deterministically reach the "period has ended" state a real clock
would eventually reach on its own:
```
UPDATE subscriptions SET current_period_end = now() - interval '1 day' WHERE id = '<sub1>'
GET /api/v1/subscription (lazy-expiry runs on this read):
  404 NOT_FOUND "no active subscription"
DB: select status from subscriptions where id='<sub1>':  EXPIRED   <- flipped by the read, not just incidentally "not entitled"
POST /api/v1/tenants ("Third Shop"):  422 VALIDATION_ERROR (blocked)
GET /api/v1/tenants/:TENANT1:  200 -- still fully there, nothing destroyed
POST /subscription (renew): 201 {status: ACTIVE, current_period_start: <now>, ...}  -- reactivated
POST /api/v1/tenants ("Third Shop") again:  201 -- succeeds now that it's renewed
```
**PASS** — expiration never destroys tenant data, correctly blocks *new*
tenant creation only, and is fully recoverable by renewing.

#### Test 7 — subscription webhook safety (no subscription Stripe events exist, but the endpoint must stay safe)

```
signed "customer.subscription.updated" event (unrelated to any payment row):
  HTTP 200 (looked up by its object id against `payments`, not found, safely ignored)
same event delivered again (duplicate):
  HTTP 200 (same safe no-op, idempotent by construction)
same event with an invalid signature:
  HTTP 401 UNAUTHORIZED "invalid Stripe signature"
```
**PASS** — confirms the existing (order-payment-only) webhook handler doesn't
crash or mis-process a subscription-shaped Stripe event; signature
verification and duplicate-delivery safety both hold for event types the
system doesn't model, exactly as they do for the ones it does.

#### Test 8 — cross-user safety

```
User B subscribes (starter-yearly), then cancels:
  DELETE /subscription:  200 {status: CANCELED, plan_id: <yearly>}
User A renews (independently, after A's own Test-7-style cancel):
  POST /subscription:  201 {id: <A's own new row>, plan_id: <A's monthly plan>, status: ACTIVE}
DB: subscriptions where user_id = A:  1 row, ACTIVE, A's own plan_id only
```
**PASS** — subscriptions are user-scoped in the schema (no tenant coupling at
all); B's cancel and A's renew operated on entirely separate rows, confirmed
via direct DB read.

#### Regression — cancel/renew cycle didn't disturb CANCEL, and prior order flow still works

```
DELETE /subscription (Test-7-in-script cancel):  200 {status: CANCELED}
POST /api/v1/tenants (new tenant while CANCELED):  422 (blocked, correct)
GET /api/v1/tenants/:TENANT1:  200 (still fine)
--- unrelated regression: fresh COD checkout end-to-end ---
POST cart/items -> POST orders -> POST orders/:id/pay {"payment_method":"COD"}:
  order status: PENDING_PAYMENT -> CONFIRMED, payment_status: PENDING (COD, correct pre-existing behavior)
```
**PASS** — the order-cancellation-lifecycle work from the previous pass and
the base checkout flow are both unaffected by this session's billing changes.

### Build/runtime verification

`payment-service`: `gofmt -l` / `go build ./...` / `go vet ./...` clean (no
test files). Migration `000005_billing_renewal` applied against the running
`payment_db` (`go run ./cmd/migration up`, executed inside the container).
Container force-recreated; confirmed clean boot (`payment-service listening
on :8080`, new route `POST
/api/v1/internal/users/:userId/subscription/mark-past-due` registered in the
Gin route dump), zero panics/fatals across the full test session.

### Remaining subscription-related gaps

- **No real recurring Stripe billing.** This was a deliberate scope decision,
  not an oversight — the task explicitly said to follow the existing manual-
  periods model rather than invent a new billing architecture if that's what
  the code already intentionally does, and this codebase's `Subscribe` has
  been stubbed-charge by design since before this pass (`billing/services`
  package doc comment). Real recurring billing would mean: a `Gateway`
  method to create a Stripe `Subscription`/`price_id`, `stripe_subscription_id`
  storage, and real `invoice.paid`/`invoice.payment_failed` webhook handling
  replacing the internal `mark-past-due` stand-in used here. Flagged in
  `GAP_ANALYSIS.md` as needing a product decision on scope; unchanged by this
  pass.
- `GET /subscription` (the caller's-own-subscription endpoint) only ever
  surfaces `ACTIVE` rows (`404` otherwise) — this was pre-existing behavior
  for `CANCELED` and is now equally true for `EXPIRED`/`PAST_DUE`. A real
  product would likely want this endpoint to show "your subscription is
  PAST_DUE / EXPIRED" rather than a bare 404; the internal entitlement route
  (`.../subscription` with `X-Internal-Key`) already returns the real status
  string correctly either way, so the tenant-creation gate itself is
  unaffected — this is a minor UX gap in the customer-facing read, not a
  correctness bug.
- `DELETE /subscription` (Cancel) is still scoped to `ACTIVE` rows only (via
  `GetActiveByUser`, pre-existing), so a `PAST_DUE` subscription can't be
  directly canceled — it can only be renewed back to `ACTIVE` or left to
  lapse to `EXPIRED`. Not exercised by the task's requirements; noted for
  completeness.

## Password / account security lifecycle — implementation + real API verification (2026-09-11)

### What already existed (unchanged)

Local email+password auth in user-management (`internal/auth`) was already
solid: bcrypt password hashing, stateless HS256 JWT access tokens (15m),
opaque refresh tokens hashed at rest with rotation + reuse detection
(`refresh_tokens` table, family-based), and a Redis session store the
`AuthGuard` checks on every request (so logout is instant on this service).
What was completely missing, confirmed by `GAP_ANALYSIS.md` (`"no route, no
service, no token table — does not exist at all"`): change-password,
forgot-password, reset-password. There was also no "revoke every session for
a user" capability anywhere — only revoke-by-id / revoke-by-family /
revoke-by-session existed, all scoped to a single session.

Also confirmed by inspection: user-management had no outbound Kafka
publisher and no mail-service HTTP client at all (`internal/events` and
`internal/infrastructure/kafka` are empty directories). mail-service is
reached directly over HTTP by other services (notification-service already
does this via its own `internal/mailclient` package hitting mail-service's
`POST /internal/mail`, X-Internal-Key-guarded) — this is the existing,
already-precedented integration pattern, not something invented for this
task.

### What was implemented

1. **`password_reset_tokens` table** (migration `000011_password_reset`) —
   single-use, short-lived (30 min), stored only as a SHA-256 hash, exactly
   mirroring `refresh_tokens.token_hash`'s existing pattern. `used_at` is set
   on consumption inside a `SELECT ... FOR UPDATE`-locked transaction (same
   row-lock pattern payment-service's webhook handler already uses), so a
   concurrent double-submit of the same token can't both succeed.
2. **`RevokeAllForUser`** added to `RefreshTokenRepository` — revokes every
   not-yet-revoked refresh token for a user (all devices/sessions) in one
   statement and returns the distinct session ids, which the service then
   deletes from Redis. This is the "revoke everywhere" primitive both
   change-password and reset-password use; nothing like it existed before.
3. **`POST /auth/change-password`** (authenticated) — requires the current
   password, rejects a new password identical to the current one, hashes the
   new one with the same bcrypt call `Register`/`Login` already use, then
   revokes every session for that user (this one included — the handler also
   clears the caller's own cookies, since their own access/refresh tokens are
   now dead too).
4. **`POST /auth/forgot-password`** (public) — always returns the same `200`
   + generic message whether or not the email belongs to an account, or
   whether it's an OAuth-only account with no password to reset. If it does
   resolve to a real password-having user, a single-use token is created and
   emailed via a new `internal/mailclient` package (byte-for-byte the same
   client shape notification-service already uses against mail-service) and
   a new `"password_reset"` template added to mail-service's existing
   template registry — no new email infrastructure, reusing exactly what was
   there.
5. **`POST /auth/reset-password`** (public) — consumes the token
   (lock-check-mark-used in one transaction), sets the new password, then
   revokes every session for that user, the same as change-password.
6. New downstream-service config for user-management:
   `services.mail_service_url` / `services.mail_internal_key`
   (`USER_MAIL_SERVICE_URL` / `USER_MAIL_INTERNAL_KEY`), mirroring the
   existing `payment_service_url`/`payment_internal_key` pattern exactly.

**Deliberate choice**: the reset-token value is never returned in any API
response (that would let anyone reset anyone's password without ever
touching their inbox). To verify the real end-to-end flow without a human
checking an inbox, this session used **Mailpit** — the dev-only SMTP catcher
already provisioned in `docker-compose.dev.yml` for exactly this purpose
(`mail-service delivers here in dev; nothing leaves the machine`, per that
file's own comment) — and read the real, delivered email's HTML body back
via Mailpit's own REST API (`GET /api/v1/messages`, `GET
/api/v1/message/:id`) to extract the real token, the same way a user would
by opening the email. No log line, no test hook, no mock was added anywhere
in the application code for this.

### Files changed

- `user-management/migrations/000011_password_reset.{up,down}.sql` (new)
- `user-management/internal/auth/domain/{password_reset.go (new), errors.go, refresh_token.go}`
- `user-management/internal/auth/repository/{password_reset_repository.go (new), refresh_repository.go}`
- `user-management/internal/auth/services/auth_service.go` — `ChangePassword`, `RequestPasswordReset`, `ResetPassword`, `revokeAllSessions`; `Service`/`New(...)` gained `mail *mailclient.Client` + `frontendURL`
- `user-management/internal/auth/handler/http/handler.go` — `ChangePassword`, `ForgotPassword`, `ResetPassword`
- `user-management/internal/auth/routes/routes.go` — wires the mail client, mounts the 3 new routes
- `user-management/internal/auth/dto/dto.go` — `ChangePasswordRequest`, `ForgotPasswordRequest`, `ResetPasswordRequest`
- `user-management/internal/httpx/errors.go` — classifies the 3 new domain errors
- `user-management/internal/mailclient/mailclient.go` (new) — mirrors `notification-service/internal/mailclient`
- `user-management/internal/config/config.go` + `configs/config.dev.yaml` + `configs/config.yaml` — mail-service URL/key
- `mail-service/internal/mail/templates/templates.go` — new `"password_reset"` template
- `deployments/compose/docker-compose.dev.yml`, `.env.dev` — `USER_MAIL_SERVICE_URL`, `USER_MAIL_INTERNAL_KEY`

### Database changes

`password_reset_tokens` table (`user_id`, `token_hash` UNIQUE, `expires_at`,
`used_at`, `created_at`), migration `000011`, applied against the running
`user_management_db`.

### API changes

- `POST /api/v1/auth/change-password` (authenticated) — `{current_password, new_password}` → `204`
- `POST /api/v1/auth/forgot-password` (public) — `{email}` → `200 {"message": "if that email is registered, a reset link has been sent"}`, always
- `POST /api/v1/auth/reset-password` (public) — `{token, new_password}` → `204`

### Webhook/event changes

None — this flow doesn't touch Kafka or the Stripe webhook at all. The only
new inter-service call is a direct, synchronous HTTP call from
user-management to mail-service's existing `POST /internal/mail`, using the
same client pattern notification-service already had in production.

### Real API scenarios executed (all against the live Docker stack)

#### A. change-password

```
A1. wrong current_password:              401 UNAUTHORIZED "current password is incorrect"
A2. new_password == current password:    422 VALIDATION_ERROR "new password must be different from the current password"
A3. no Authorization header:             401 UNAUTHORIZED "authentication required"
A4. correct current + valid new:         204
A5. login with the OLD password:         401 "invalid credentials"
A6. login with the NEW password:         200, new tokens issued
A7. the PRE-CHANGE access token on /auth/me:   401 "session is no longer valid"
A8. the PRE-CHANGE refresh token on /auth/refresh: 401 "refresh token is invalid or expired"
```
**PASS** — current-password verification, reuse rejection, unauthenticated
rejection, and full session invalidation (both the access token's Redis
session and the refresh token) all confirmed for real.

#### B. forgot-password / reset-password

```
B1. forgot-password, UNKNOWN email:   200 {"message":"if that email is registered, a reset link has been sent"}
B2. forgot-password, KNOWN email:     200 {"message":"if that email is registered, a reset link has been sent"}   <- byte-identical to B1
--- real email fetched from Mailpit ---
Subject: "Reset your password"
real token extracted from the email's own reset link: W5FsTJ-kWLD8j86Lmwl05r5Ix6TYvd9s8rGvC7m74T0
B3. reset-password, a MADE-UP token:  400 BAD_REQUEST "invalid or expired reset token"
B4. reset-password, the REAL token:   204
B5. login with the OLD password:      401 "invalid credentials"
B6. login with the NEW password:      200, new tokens issued
B7. REUSE the same (now-consumed) token: 400 BAD_REQUEST "invalid or expired reset token"
```
**PASS** — unknown vs. known email produce an identical response (no account
enumeration), the real single-use token round-tripped through a real email
works exactly once, and a second use of the same token is rejected.

#### B (part 2) — session revocation on reset, expired token, cross-user isolation

```
User C logs in FIRST (holds a pre-reset access+refresh token pair), THEN
requests+uses a real reset token:
  reset-password: 204
  pre-reset access token on /auth/me:        401 "session is no longer valid"
  pre-reset refresh token on /auth/refresh:  401 "refresh token is invalid or expired"

EXPIRED token (forced via direct DB update on that one token's expires_at,
the same fault-injection technique used throughout this whole test session --
there's no way to wait out a real 30-minute TTL here):
  reset-password with the expired token:  400 BAD_REQUEST "invalid or expired reset token"
  user D's ORIGINAL password still works: 200   <- expired-token attempt had zero effect

CROSS-USER: user C's new password works, user D's password is completely
unaffected by C's (or anyone else's) reset — each token only ever resolves
to the one user_id it was issued for.
```
**PASS** — reset-password revokes every existing session for the account
(not just issuing a new one), an expired token is inert, and one user's
reset can never touch another user's credentials.

#### C. Regression — register/login/refresh/logout still work

```
register:            success
login:                200
refresh (rotation):   200, new pair issued
logout:                204
/auth/me after logout: 401 "session is no longer valid"
```
**PASS** — none of the existing auth lifecycle behavior changed.

### Build/runtime verification

`user-management` + `mail-service`: `gofmt -l` / `go build ./...` / `go vet
./...` clean (no test files). Migration `000011_password_reset` applied
against the running `user_management_db`
(`docker exec user-management go run ./cmd/migration up`). Both containers
force-recreated; confirmed clean boot (`user-management`: `Server is running
on port 8080`, all 3 new routes present in the Gin route dump;
`mail-service`: `mail-service listening on :8080`, `mail-service consuming
order-events, payment-events` unchanged), zero panics/fatals across the full
test session.

### Remaining limitations / deliberate scope decisions

- **Cross-service revocation is still access-token-TTL-bounded on every
  *other* service.** Exactly as `GAP_ANALYSIS.md` already documented for
  logout: revoking sessions on user-management is instant (Redis session
  deleted, checked on every request to user-management itself), but a
  leaked/already-issued access token for another service (order, product,
  cart, inventory, payment, notification) is a stateless JWT those services
  verify locally with no revocation check — it keeps working there until it
  naturally expires (≤15 min). Password change/reset close this exposure on
  user-management immediately and everywhere else within one access-token
  TTL; making it instant everywhere would mean adding a shared revocation
  check to every service's `AuthGuard` (a real architecture trade-off flagged
  by the gap analysis as a follow-up decision, not something to silently
  add here).
- No password-history table — only the *current* password is checked for
  reuse on change-password, per the task's own scope ("prevent reuse of the
  current password", not a history). Not implemented for reset-password
  (the requester has presumably forgotten their password, so there's nothing
  meaningful to compare against beyond the current hash, which reset already
  overwrites).
- No rate-limiting was added to `forgot-password` specifically (the service
  has no existing rate-limit middleware to hook into beyond the global one
  already applied service-wide) — worth a follow-up if abuse becomes a
  concern, not implemented here to avoid inventing new infrastructure.
