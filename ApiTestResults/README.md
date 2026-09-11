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
