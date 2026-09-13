# Gap Analysis — ecommerce-sas (Phases 1–4)

Full-repo inspection (all 8 services, all migrations, all routes) against the P0/P1/P2
list. Verdicts: **WORKING** / **PARTIAL** / **MISSING** / **BROKEN**.

> **Update 2026-09-11**: P0 items 1–3 and P1 items 4–6 (§8, fix-order items 1–6)
> are now implemented and verified through the real running services — see
> `ApiTestResults/README.md` § "Gap-analysis fixes — implementation + real API
> verification". The verdicts and code excerpts below are left as they were at
> the time of the original audit (a historical record of what was found); they
> no longer describe the current behavior for those six items. Everything from
> fix-order item 7 onward is still open.

Architecture confirmed and preserved: Go + Gin + `database/sql` (no GORM), one
Postgres DB per service, clean-architecture-ish module layout
(`domain/dto/repository/services/handler/routes`), tenant membership is
`memberships(tenant_id, user_id, role_id)` — **no `user_role` table, and none is
needed** (see §2).

---

## 0. Executive summary

The platform is in noticeably better shape than the task brief assumed — most of
"P0" is already correctly implemented (refresh-token rotation+reuse-detection,
multi-tenant RBAC, transactional tenant-creation-makes-ADMIN, atomic coupon
redemption, row-locked inventory reservation, reserve-before-order-create). The
real gaps cluster into a smaller, sharper set:

1. **No cross-service session/token revocation** — logout is instant on
   user-management but a leaked access token still works on every other service
   until it naturally expires (≤15 min).
2. ✅ **DONE 2026-09-11** — ~~No password reset / change-password flow at
   all~~ — implemented; see fix-order item 9 (§8) and
   `ApiTestResults/README.md` § "Password / account security lifecycle".
3. **Payment webhook has a genuine concurrency race** — no row lock, so two
   truly-concurrent deliveries of the same Stripe event can both pass the
   "already processed" guard and double-publish `payment.captured` (duplicate
   emails/notifications). Sequential retries are fine.
3b. **Order status is never synced from the webhook at all** — payment-service
   publishes `payment.captured` to Kafka, but order-service has no consumer
   wired up (the `Consumer` type exists, unused — `grep` confirms zero call
   sites). An order only leaves `PENDING_PAYMENT` when the customer's browser
   calls `POST /orders/:id/pay` **again**; `GET /orders/:id` never triggers a
   live Stripe check either. A captured card payment for a customer who never
   returns to the site leaves the order stuck `PENDING_PAYMENT` forever.
3c. **A failed CARD payment does not release its inventory reservation** — only
   order *cancellation* releases stock; `Pay()` returning `Status=FAILED` just
   falls through, leaving the reservation in place indefinitely until a human
   explicitly cancels the order. Reserved-but-unsellable stock can accumulate
   from abandoned/declined card attempts.
4. **Coupon discount vs. redemption-count is not atomic with order creation** —
   if `RedeemCoupon` fails after the order is written, the customer keeps the
   discount but the counter never increments (silently logged, not surfaced).
5. **Returns don't check previously-returned quantity** — a return can be
   requested for a SKU up to its *original order quantity* even if an earlier
   return for that SKU on the same order was already approved and refunded, and
   there's no de-dup against a REQUESTED-but-not-yet-resolved return either.
6. ✅ **DONE 2026-09-11** — ~~No subscription renewal or ongoing
   enforcement~~ — renewal (with idempotent no-double-extend), a `PAST_DUE`
   grace state for failed renewal charges, and lazy expiry are now
   implemented; see fix-order item 11 (§8) and
   `ApiTestResults/README.md` § "Subscription lifecycle". Ongoing
   per-operation enforcement on an existing tenant after its owner's
   subscription lapses was a deliberate **non**-change — see that section for
   the reasoning (tenant-creation stays the only gate, matching this
   codebase's existing separation of entitlement from tenant data).
7. **Repository layer defense-in-depth gap (not a live exploit today)** — several
   mutation queries filter by `id` only, relying on the service layer to have
   pre-verified tenant ownership. Every currently-reachable HTTP path is safe,
   but the SQL itself doesn't self-defend. Detailed list in §3.
8. **Product hard-delete has no safety rail** — `DELETE /products/:id` is a
   real `DELETE FROM products`, cascading away variants/images/specs/reviews,
   with no check for order history.
9. **No purchase-eligibility check on reviews** — any authenticated user can
   review any product without ever buying it.
10. Several smaller items below (invite-by-email doesn't exist — by design,
    membership requires an existing user id; no "list roles" for tenant
    admins; no primary-image flag; minor semantics on order.payment_status
    after a partial-refund return).

Nothing here requires touching the architecture, adding GORM, or introducing a
`user_role` table. Every fix below is additive/local to existing
repositories/services.

---

## 1. Database audit

### user-management (`user_management_db`)

| Table | PK | FK | Unique | tenant_id? | Notes |
|---|---|---|---|---|---|
| `users` | `id` | — | `email` (CITEXT) | n/a (global) | `password_hash` nullable (OAuth), `is_super_admin`, `phone` (added `000010`) |
| `roles` | `id` | — | `name` | n/a (global) | CHECK `name IN ('SUPER_ADMIN','ADMIN','MANAGER','CUSTOMER')`, seeded |
| `permissions` | `id` | — | `name` | n/a (global) | fine-grained permission catalog |
| `role_permissions` | `(role_id,permission_id)` | →roles CASCADE, →permissions CASCADE | — | n/a | role → baseline permission set |
| `tenants` | `id` | `owner_user_id`→users RESTRICT | `slug` (CITEXT) | **is** the tenant boundary | `status` CHECK ACTIVE/SUSPENDED/INACTIVE (SUSPENDED never actually set anywhere) |
| `memberships` | `id` | →tenants CASCADE, →users CASCADE, →roles RESTRICT | `(tenant_id,user_id)` | ✅ | **this is the "tenant_memberships" table** — one row per (user,tenant) with its own `role_id`, so a user can be ADMIN in tenant A and MANAGER in tenant B simultaneously. Confirmed by code + constraint. |
| `membership_permissions` | `(membership_id,permission_id)` | →memberships CASCADE, →permissions CASCADE | — | via membership | per-member permission override, on top of role baseline |
| `refresh_tokens` | `id` | →users, session grouping via `family_id`/`session_id` | `token_hash` | n/a | opaque token hashed at rest (SHA-256); rotation + reuse-detection implemented |
| `addresses` | `id` | →users CASCADE | one default-shipping / one default-billing per user (partial unique idx) | n/a | `GetByID/Update/Delete` filter by `id` only, service layer compares `UserID` — same "compensate in app code" pattern as flagged in §3 |
| `subscriptions`, `plans` | — | — | — | — | **dropped from this DB** (migration `000009_drop_billing`) — billing now lives entirely in payment-service |

**No `user_role` table exists, and per the schema above none should be added** — role
is already correctly modeled per-membership (per tenant), which is exactly the
"User A → Tenant1=ADMIN, Tenant2=MANAGER" requirement.

### product-service (`product_db`)

| Table | PK | FK | Unique | tenant_id | Notes |
|---|---|---|---|---|---|
| `categories` | `id` | `parent_id`→self SET NULL | `(tenant_id,slug)` | ✅ | self-referencing tree |
| `products` | `id` | — | `(tenant_id,slug)` | ✅ | generated `search tsvector` + GIN index |
| `product_variants` | `id` | →products CASCADE | `(tenant_id,sku)` | ✅ (denormalized) | one `is_default` per product (partial unique idx); `price_cents CHECK >= 0` |
| `product_images` | `id` | →products CASCADE | `storage_key` | — (via product) | no `is_primary` flag (position=0 convention only) |
| `product_categories` | `(product_id,category_id)` | both CASCADE | — | via product | M:N join |
| `product_attributes` | `id` | →categories CASCADE | `(category_id,code)` | ✅ | `role` CHECK VARIANT/SPEC |
| `product_attribute_values` | `id` | →product_attributes CASCADE | `(attribute_id,value)`, `(attribute_id,id)` | via attribute | the `(attribute_id,id)` unique lets `product_variant_options` FK safely |
| `product_specs` | `(product_id,attribute_id)` | both CASCADE | is the PK | via product | display-only spec values |
| `product_variant_options` | `(variant_id,attribute_id)` | →variants CASCADE, composite FK →(attribute_id,attribute_value_id) | is the PK | via variant | DB-enforced "value belongs to that attribute" |
| `product_reviews` | `id` | →products CASCADE | `(product_id,customer_id)` | ✅ | one review per customer per product (UPSERT) |

### inventory-service (`inventory_db`)

| Table | PK | FK | Unique | tenant_id | Notes |
|---|---|---|---|---|---|
| `stock_items` | `id` | — | `(tenant_id,sku)` | ✅ | `on_hand`, `reserved`; `Available() = on_hand - reserved`; every mutation goes through `LockBySKU` (`SELECT ... FOR UPDATE`) inside a transaction — **correct, safe under concurrency** |
| `stock_movements` | `id` | →stock_items | — | via item | audit ledger, `type` (RECEIVE/RESERVE/RELEASE/SHIP/ADJUST) |

Clean — no tenant-scoping findings anywhere in this service.

### cart-service (`cart_db`)

| Table | PK | FK | Unique | tenant_id | Notes |
|---|---|---|---|---|---|
| `carts` | `id` | — | one ACTIVE cart per `(tenant_id,customer_id)` (partial unique idx) | ✅ | `customer_id` always from JWT, never a path/body param |
| `cart_items` | `id` | →carts CASCADE | `(cart_id,sku)` | via cart | price/name **snapshotted** at add-time |

### order-service (`order_db`)

| Table | PK | FK | Unique | tenant_id | Notes |
|---|---|---|---|---|---|
| `orders` | `id` | — | `(tenant_id,customer_id,idempotency_key)` partial unique (key IS NOT NULL) | ✅ | money breakdown columns, `shipping_address`/`billing_address` JSONB snapshots, `tracking_carrier/number`; **6 mutation methods filter by `id` only** (see §3 Tier 1) |
| `order_items` | `id` | →orders CASCADE | — | via order | full line snapshot (sku/price/name) — no FK to `products`, by design |
| `order_config` (tax) | `tenant_id` (PK) | — | is the PK | ✅ | `tax_rate_bps CHECK 0–10000`, `tax_inclusive` |
| `shipping_rates` | `id` | — | — | ✅ | flat + `free_over_cents` |
| `coupons` | `id` | — | `(tenant_id,code)` | ✅ | `redeemed_count` — **global counter only, no per-user tracking, no `coupon_redemptions` table** |
| `order_returns` | `id` | →orders CASCADE | — | ✅ | `Resolve` filters by `id` only (§3) |
| `order_return_items` | `id` | →order_returns CASCADE | — | via return | |

### payment-service (`payment_db`)

| Table | PK | FK | Unique | tenant_id | Notes |
|---|---|---|---|---|---|
| `payments` | `id` | — | — | ✅ | `refunded_cents CHECK 0 ≤ x ≤ amount_cents`; `Update` filters by `id` only (§3); `HandleWebhook` reads via `GetByGatewayRef` (no lock — see §Finding P0-4) |
| `plans` | `id` | — | `code` | **none — correctly global**, platform-wide catalog | |
| `subscriptions` | `id` | →plans RESTRICT | one ACTIVE per `user_id` (partial unique idx) | **none — correctly user-scoped, not tenant-scoped** | confirmed deliberate: a user subscribes before any tenant exists |

### mail-service / notification-service

`mails` — no `tenant_id` (by design, internal-only, keyed by recipient address).
`notifications` — `tenant_id` nullable, **not** filtered by tenant in
list/mark-read queries (feed is intentionally "all my notifications across every
tenant I'm in" — flagged as a UX/design question in §3, not a security bug,
since these routes take no `:tenantId` path param at all).

---

## 2. Tenant membership / RBAC model — confirmed design

```
memberships (id, tenant_id, user_id, role_id, UNIQUE(tenant_id, user_id))
   → roles (SUPER_ADMIN | ADMIN | MANAGER | CUSTOMER)
   → membership_permissions (per-member fine-grained override, additive to role_permissions)
```

- **Confirmed**: the same user can hold different roles in different tenants —
  one `memberships` row per `(tenant_id, user_id)`, each with its own `role_id`.
  This already satisfies "User A → Tenant1=ADMIN, Tenant2=MANAGER, Tenant3=CUSTOMER."
- **Tenant creation → creator becomes ADMIN**: transactional
  (`tenant/services/service.go` `Create()` wraps the tenant INSERT and the
  membership INSERT in one `platform.RunInTx`). **WORKING.**
- **Membership API — all present and correctly guarded**: list members
  (ADMIN/MANAGER), add member (ADMIN), change role (ADMIN), remove member
  (ADMIN), list a member's permissions / grant / revoke permission override
  (ADMIN). Cross-tenant modification is correctly blocked — the guard resolves
  the caller's *own* membership row for the `:tenantId` in the path, so an
  ADMIN of tenant A gets `403` calling any member-management endpoint under
  tenant B (no membership row found for B). **WORKING.**
- **Authorization has two layers**, both implemented: `RequireTenantRole`
  (coarse, role-name based — this is what every route currently uses) and
  `RequirePermission` (fine-grained, unions role-baseline + per-member grants —
  fully implemented, including the guard function, but **no route currently
  uses it**). **PARTIAL** — infrastructure done, not wired to any endpoint yet.
  Not a defect; just unused capability.
- **CUSTOMER is a real seed row** in `roles`, assignable via `AddMember`/`ChangeRole`
  — but nothing in user-management ever auto-creates a CUSTOMER membership row
  for a shopper (only the ADMIN-on-tenant-creation path exists). Whether a
  "customer" should get an explicit membership row when they place their first
  order in a tenant, or whether "any authenticated user" is implicitly the
  customer role for storefront actions (which is how cart/order/review
  endpoints actually work today — they only check `guards.Authenticated`, not
  tenant membership), is a **product decision**, not a bug — flagging for your
  call, not assuming.
- **No `GET /roles` usable by a tenant ADMIN** — the role catalog endpoint is
  super-admin-only. A tenant admin adding a member has to hardcode
  `MANAGER`/`CUSTOMER` client-side (matches the DTO's `oneof` binding). Minor —
  **PARTIAL**.
- **Invite-by-email does not exist** — `AddMember` requires an existing
  `user_id` (UUID), FK-enforced. No pending-invite state/table/email dispatch.
  **MISSING**, and worth confirming you actually want it before building it —
  it's a real feature addition (invite record, email, accept flow), not a small
  gap.

---

## 3. Cross-tenant isolation — security audit

**No live, HTTP-reachable cross-tenant exploit was found.** Every endpoint that
takes `:tenantId` + a resource id in the path currently gets its final safety
from the service layer re-fetching the resource with a tenant-scoped query (or
comparing `.TenantID`/`.ProductID` fields) before acting — so a valid member of
tenant B supplying a tenant-A resource id today gets `404`/`403`, not access.

That said, this safety lives **entirely in application code, not in the SQL**,
for the following mutation methods — the repository query alone would happily
touch another tenant's row if any future code path (a new endpoint, a webhook
handler, a refactor) calls it without re-verifying tenancy first. Fixing these
(adding `tenant_id = $N` to the WHERE clause) is cheap and removes the risk
permanently — recommended as part of P0 cleanup even though nothing is
exploitable today:

**Tier 1 — financial/privilege tables, zero SQL-level tenant defense:**
- `order-service`: `orders.AttachPayment/MarkConfirmed/MarkPaymentPaid/MarkPaymentRefunded/MarkFulfilled/MarkCancelled` — all filter by `id` only (`internal/order/repository/repository.go:226-285`)
- `payment-service`: `payments.Update` — filters by `id` only (`internal/payment/repository/repository.go:89-107`)
- `order-service`: `order_returns.Resolve` — filters by `id` only (`internal/returns/repository/repository.go:173-182`)
- `user-management`: `memberships.GetByID/UpdateRole/Delete` — filter by `id` only (`internal/membership/repository/repository.go:66-138`); `GetByID` is currently dead code (unused)
- `user-management`: permission-grant queries (`AssignToMembership`, `RevokeFromMembership`, `EffectiveForMembership`) — scoped by `membership_id` only, no tenant check in SQL
- `product-service`: `product_variants.GetByID/Update/Delete/ClearDefault` — filter by `id` only (`internal/product/repository/variant_repository.go`)
- `order-service`: `coupons.Redeem` — filters by `id` only (only ever called with a tenant-verified coupon id, but the query itself doesn't check)

**Tier 2 — lower severity (no `tenant_id` column on the table at all, missing even parent-id scoping):**
- `product-service`: `product_images.GetByID/Update/Delete` — filter by `id` only, not even `product_id`
- `cart-service`: `carts.Touch`, `cart_items.SetQuantity` — filter by `id` only; not reachable by an attacker since the id always comes from the caller's own JWT-derived cart

**Recommendation**: add the tenant/parent filter to every query above. This is a
mechanical, low-risk change (same pattern already used correctly by the
majority of the codebase's repositories) and should be done as part of P0.

**Design note, not a bug**: notification list/mark-read queries filter by
`user_id` only (no tenant scoping) — by design, since those routes take no
`:tenantId` path param (the feed is deliberately "everything across every
tenant I belong to"). Confirm this is the intended UX before leaving it as-is.

---

## 4. P0 — detailed findings

### 4.1 Authentication lifecycle

| Capability | Status | Notes |
|---|---|---|
| Register / Login | **WORKING** | bcrypt, duplicate-email → 409 via DB unique constraint |
| Access token (JWT HS256, 15m) | **WORKING** | shared secret, verified locally by every service |
| Refresh (rotation) | **WORKING** | opaque token, hashed at rest, rotated every use, in a DB tx |
| Refresh reuse detection | **WORKING** | reused/revoked token → revokes entire token family + Redis session (stolen-token response) |
| Refresh with invalid/expired token | **WORKING** | 401, `ErrRefreshInvalid` — but expired vs. reused aren't distinguishable from the response body (minor) |
| Logout | **PARTIAL** | fully revokes session + refresh family on user-management; but access tokens are stateless JWTs verified locally by every *other* service with no revocation check — a leaked access token still works elsewhere for up to 15 min after logout |
| Protected endpoint after logout | **PARTIAL** | 401 immediately on user-management itself; still valid (until natural expiry) on order/product/cart/inventory/payment/notification |
| Password validation | **WORKING (minimal)** | length 8–128 only, no complexity rule — acceptable but worth a decision, not a bug |
| Duplicate email | **WORKING** | 409 |
| Password reset / change password | **MISSING** | no route, no service, no token table — does not exist at all |

**Fix plan**: (a) password reset/change-password flow (new, real feature — token
table + email via existing mail-service + endpoints, following the exact
pattern of `refresh_tokens`). (b) Optional hardening for cross-service
revocation: either shorten access-TTL further, or add a lightweight
Redis-backed revocation check to the shared `middleware.AuthGuard` used by
every service (adds one Redis round-trip per request — a real architecture
trade-off, not a "just do it" — will present as a P0-followup decision rather
than silently implementing).

### 4.2 Tenant membership & role management

All core capabilities **already WORKING** (see §2) — list/add/change-role/remove
member, permission overrides, cross-tenant isolation on membership endpoints.
Gaps: no tenant-facing "list roles" (PARTIAL), no invite-by-email (MISSING, and
a real feature decision).

### 4.3 Cross-tenant isolation

See §3. No live exploit; defense-in-depth gap in ~12 repository methods across
5 services. **Recommended P0 fix**: add tenant_id (or parent-id) to every WHERE
clause listed.

### 4.4 CARD payment completion

| Capability | Status | Notes |
|---|---|---|
| PaymentIntent creation → client_secret | **WORKING** | confirmed live in this session's test run — real `pi_...` id, order stays `PENDING_PAYMENT` |
| Webhook signature verification | **WORKING** | HMAC-SHA256 over `timestamp.payload`, constant-time compare (`hmac.Equal`); invalid signature → 401. **Minor gap**: no timestamp-tolerance/replay window check (Stripe best practice) |
| Webhook → payment status update | **WORKING** | `payment_intent.succeeded` → captured; `payment_intent.payment_failed`/`canceled` → failed |
| Webhook → order status update | **MISSING** | payment-service publishes `payment.captured` to Kafka, but **order-service has no consumer for it at all** — the `events.Consumer` type is defined (identical boilerplate to every other service) but never instantiated or run anywhere in order-service (confirmed: zero call sites). The order row only leaves `PENDING_PAYMENT` when something calls `POST /orders/:id/pay` again (→ `Sync` → live Stripe check). `GET /orders/:id` is a plain DB read — it never triggers a re-sync. **A card payment captured by webhook while the customer never returns to the site leaves that order stuck at `PENDING_PAYMENT` forever.** This is the single most important P0 fix: add an order-service Kafka consumer for `payment.captured` / `payment_intent.payment_failed`, mirroring the exact pattern mail-service/notification-service already use. |
| Webhook idempotency (sequential) | **WORKING** | state-guard `if p.Status != Pending { return nil }` — second delivery of an already-processed event is a safe no-op |
| Webhook idempotency (concurrent) | **BROKEN** | `HandleWebhook` (`payment/services/service.go:186-210`) does a plain `SELECT` (`GetByGatewayRef`, no lock) then a plain `UPDATE` — **not** wrapped in `RunInTx` + `SELECT...FOR UPDATE` the way inventory-service correctly does it. Two genuinely concurrent deliveries of the same event can both pass the `Status != Pending` guard before either commits, both call `MarkCaptured` + `Update` + `publish(PaymentCaptured)` — the DB row ends up correct (idempotent field values) but **the Kafka event double-publishes**, causing duplicate receipt emails and duplicate notifications. |
| Unknown-order webhook | **WORKING** | `GetByGatewayRef` miss → `return nil` → 200 OK (correctly avoids Stripe retry-storming a webhook that will never resolve) |
| Already-paid order webhook | **WORKING** | same state-guard as above |
| Failed / cancelled payment | **WORKING** | maps to `StatusFailed`, order stays `PENDING_PAYMENT` (customer can retry `Pay`) |
| Do-not-mark-paid-on-intent-creation | **WORKING** | confirmed — `Pay` only flips to CONFIRMED when `payment.Status == "CAPTURED"`, never on intent creation |

**Fix plan**: (1) add the order-service Kafka consumer for `payment.captured`/
`payment_intent.payment_failed` — same `events.NewConsumer(...).Run(...)`
pattern already used by mail-service/notification-service, just wired to
update `orders.status`/`payment_status` instead of sending mail. (2) wrap
`HandleWebhook`'s read+guard+update in `platform.RunInTx` +
`SELECT ... FOR UPDATE` on the payment row — exact same pattern already used
correctly in inventory-service's `mutate`/`mutateBulk`. Both are small, local,
consistent-with-existing-patterns fixes. Optionally add the Stripe
timestamp-tolerance check.

### 4.5 Inventory reservation lifecycle

| Capability | Status | Notes |
|---|---|---|
| Checkout reserves inventory synchronously | **WORKING** | `Checkout()` calls `s.inventory.Reserve(...)` before the order row is even built, and aborts checkout if it fails |
| Atomic under concurrency (row lock) | **WORKING** | `LockBySKU` = `SELECT ... FOR UPDATE` inside `platform.RunInTx`, both for single (`mutate`) and bulk (`mutateBulk`) moves — the exact right pattern, already in place |
| Two customers, last unit | **WORKING** (by construction) | second reserve blocks on the row lock, then correctly fails `ErrInsufficientStock` once it sees the first's committed reservation |
| Release on order-create failure | **WORKING** | `Checkout` explicitly calls `inventory.Release` if `repo.Create` fails |
| Release on cancellation | **WORKING** | `Cancel()` calls `inventory.Release` after `MarkCancelled` succeeds |
| Release on payment FAILURE (no cancellation) | **MISSING** | `Pay()` returning `Status=FAILED` does not release the reservation — it just falls through, leaving stock reserved with no automatic path back to available until a human explicitly calls `Cancel`. Real leak risk from abandoned/declined card attempts. |
| Convert reservation → on-hand reduction on fulfil | **WORKING** | confirmed in this session's live test: `Fulfil` calls `inventory.Ship`, `on_hand` dropped, `reserved` cleared |
| Idempotent checkout retry doesn't double-reserve | **PARTIAL** | steady-state retries (first request already committed) correctly skip `Reserve` entirely — the idempotency-key check runs before it. But if a retry genuinely races the still-in-flight first request, **both** reach `Reserve` (transient double-reservation); the loser's `repo.Create` hits the unique-index violation and it calls `inventory.Release` to compensate — but that release call is fire-and-forget (`_ = s.inventory.Release(...)`), so if it itself fails (e.g. a network blip), the loser's reservation leaks permanently with no order to account for it. |
| Insufficient stock at checkout | **WORKING** | `ErrInsufficientStock` → mapped by httpx to a client error |

This item is essentially **already fully implemented** — better than the task
brief assumed. No changes recommended beyond the general Tier-1 tenant-scoping
cleanup in §3 (inventory-service itself had zero findings there).

### 4.6 Order cancellation

| Capability | Status | Notes |
|---|---|---|
| Customer cancels own eligible order | **WORKING** | |
| Seller/admin cancellation | **WORKING** | same endpoint — `load()` allows the owner OR a tenant ADMIN/MANAGER |
| Cancel after fulfilment fails | **WORKING** | DB-guarded: `MarkCancelled` only succeeds `WHERE status IN ('PENDING_PAYMENT','CONFIRMED')` |
| Refund when applicable | **WORKING** | refunds only if `PaymentStatus == PAID` — correct for COD (never PAID pre-fulfilment, so no refund attempted, correctly) |
| COD cancellation | **WORKING** | no money was ever collected pre-fulfilment, nothing to refund |
| CARD cancellation | **PARTIAL** | correctly refunds a captured payment; **does not void/cancel the Stripe PaymentIntent** if cancellation happens while the payment is still `PENDING_PAYMENT` (waiting on the customer to confirm/webhook) — a late-arriving webhook after cancellation could still capture money for a cancelled order. Edge case, not exercised by the existing test flow. |
| Inventory release | **WORKING** | |
| Notification | **WORKING** | Kafka `order.cancelled` event published |
| Duplicate cancellation request | **WORKING** | DB status guard makes the second call fail with `ErrInvalidTransition`, no double-release/double-refund |

**Fix plan**: on cancel, if `order.PaymentID != nil && PaymentStatus == PENDING`
(CARD, not yet captured), also call a new payment-service "cancel intent"
endpoint (Stripe `payment_intents/cancel`) so a late webhook can't capture a
cancelled order. Small, additive.

---

## 5. P1 — detailed findings (condensed; full detail available on request)

| # | Area | Status | Key finding |
|---|---|---|---|
| 7 | Cart lifecycle | **WORKING**, one **MISSING** | add/update/remove/clear all correct, same-SKU merges quantity, stock checked (advisory, non-reserving) at add/update time via inventory-service, product/variant must be ACTIVE to add. **Gap**: no revalidation at checkout that a SKU added earlier is still active/exists — checkout only reads the cart's price/name snapshot, never re-checks the catalog. Low risk (inventory check still gates it) but worth a decision. |
| 8 | Subscription lifecycle | **PARTIAL** | Subscribe/Get/Cancel all real (not stubs); `DELETE /subscription` genuinely cancels. Expiry is enforced *incidentally* (`IsEntitled` checks `current_period_end` every time it's called) even though the row's `status` never actually flips to `EXPIRED`. Duplicate-ACTIVE prevented by a DB partial-unique index (safe under concurrency). **MISSING**: renewal (no cron/worker — `cmd/worker` is an empty scaffold; Stripe subscription billing/webhooks not implemented, only one-off order payments are); **MISSING**: ongoing enforcement — an existing tenant keeps working forever after its owner's subscription lapses/cancels, because the check only runs once at tenant-creation. |
| — | Coupon atomicity | **WORKING** (redemption counter), **BROKEN** (order/counter consistency) | `Redeem`'s `UPDATE ... WHERE redeemed_count < max_redemptions` is correctly atomic under concurrency. But it's called *after* the order is already committed with the discount applied, and its error is only logged — a failed redemption leaves the order discounted with the counter never incremented. No per-user redemption limit exists (global counter only). |
| 9 | Product/category/variant lifecycle | **WORKING**, one **MISSING** guard | Full CRUD exists for all of categories/attributes/products/variants; SKU uniqueness and price `CHECK >= 0` are DB-enforced; publish/unpublish/archive is unified via `PATCH {status}` (no separate endpoints needed). **Gap**: `DeleteProduct` is a hard `DELETE`, cascading variants/images/specs/reviews away, with no check for existing order history (order_items don't FK to products, so nothing stops it, and nothing warns the seller). |
| 10 | Product images | **WORKING** | full presigned-upload → confirm → update(reorder via `position`) → delete lifecycle already exists on S3/MinIO (`internal/infrastructure/s3`), content-type validated, storage-key ownership checked. **Minor gap**: no explicit `is_primary` flag (relies on `position` convention). |
| 11 | Inventory management | **WORKING** | adjust/receive/reserve/release/ship all exist, all transactional + row-locked; `on_hand`/`reserved` never go negative (guarded in the domain `apply` functions); duplicate SKU registration blocked by `(tenant_id,sku)` unique constraint; unknown SKU → 404. |
| 12 | Return/refund lifecycle | **PARTIAL/BROKEN** | full return, partial return, rejection, and double-resolve prevention (`status != REQUESTED` guard) all work. **Real gap**: no check against *previously returned* quantity for a SKU on the same order — a second return request for an already-fully-returned line succeeds as long as it's ≤ the *original* order quantity; also no prevention of two simultaneously-REQUESTED returns for the same items. Minor: `orders.payment_status` flips straight to `REFUNDED` on any partial-return refund (no `PARTIALLY_REFUNDED` state). |
| 13 | Coupon edge cases | **WORKING** (mostly) | expired/not-yet-active/usage-limit/min-order/inactive all checked by the same `Redeemable()` domain method used by both the preview endpoint and real checkout (not a simplified duplicate, as the brief assumed). Concurrent redemption is safe (atomic UPDATE). **MISSING**: per-user usage limit (no redemption-log table, global counter only). |
| 14 | Shipping | **WORKING** | create/update/delete/free-shipping-threshold all implemented and correctly tenant-scoped; "invalid destination" and "multiple shipping methods" aren't modeled as separate concepts (a shipping rate is currently destination-agnostic) — confirm if that's intentional scope or a wanted feature before building it. |
| 15 | Notifications | **WORKING** | list, mark-one-read, mark-all-read, unread-count all exist; Kafka consumer retries 3× with backoff and commits-on-give-up (won't wedge the consumer group, but a given-up message is lost, not DLQ'd — see P2). |
| 16 | Reviews | **PARTIAL** | create (upsert = update-in-place), delete (author-only), public listing with aggregate summary all work; duplicate review is structurally impossible (one row per `(product,customer)`, UPSERT). **MISSING**: no purchase/fulfilment eligibility check — any authenticated user can review any product without ever having ordered it. Implementing "verified purchase" needs a cross-service decision (order-service → product-service check, or an event-driven eligibility flag) — flagging for your call rather than assuming the design. |

---

## 6. P2 — quick pass

| Item | Status |
|---|---|
| Pagination | **WORKING** — `limit`/`offset` on every list endpoint, clamped, consistent |
| Filtering/sorting | **PARTIAL** — status/category/search filters exist on the endpoints that need them; no generic sort parameter anywhere (all lists have a fixed `ORDER BY`) |
| Consistent error envelope | **WORKING** — identical `{status:"error",error:{service,code,message,method,path,request_id,timestamp,fields}}` shape everywhere |
| Request/correlation ID | **WORKING** — `RequestID()` middleware on every service, echoed in error responses; **not currently propagated across service-to-service HTTP calls** (each hop gets a fresh id) — would need to be forwarded as a header to get true cross-service correlation |
| Structured logging | **PARTIAL** — consistent `log.Printf` prefixed by service/module, not JSON-structured |
| Health checks | **WORKING** — `/api/v1/health` on all 8 services |
| Graceful shutdown | need to verify `server/shutdown.go` per service — likely present given the established pattern (context-based shutdown was part of the original scaffold); not re-audited in this pass |
| Kafka retry/DLQ | **PARTIAL** — 3 retries + 2s backoff, but a given-up message is only logged and committed, not sent to a dead-letter topic — it's simply lost |
| Redis failure behavior | not audited this pass — session store is a hard dependency for user-management auth (Redis down = login/refresh/logout all fail); worth a resilience decision |
| DB connection failure | not audited this pass — standard `database/sql` pool behavior (retries via driver, no circuit breaker) |
| Rate limiting | **WORKING** — per-IP token bucket middleware already on every service (seen earlier: 20–30 req/s) |
| Docker prod config | `docker-compose.prod.yml` exists but was not diffed against dev in this pass |

---

## 7. What I will NOT build (explicit non-recommendations)

- **No `user_role` table** — the membership model already correctly supports
  per-tenant roles; adding one would be redundant and wrong.
- **No GORM, no repository-pattern rewrite** — every fix above is a localized
  SQL/WHERE-clause or transaction-wrapping change inside existing repository
  files, using patterns already established elsewhere in the same codebase
  (e.g. copying inventory-service's `LockBySKU` pattern into payment-service's
  webhook handler).
- **No distributed locking** — every concurrency issue found (or already
  correctly solved) is a single-database-transaction problem; Postgres
  `SELECT ... FOR UPDATE` is sufficient and is the pattern to keep using.
- **No test files** — per your instruction, all of the above will be verified
  by extending `ApiTestResults/README.md` with new scripted request/response
  runs, not `*_test.go` files.

---

## 8. Proposed fix order (pending your go-ahead)

1. ✅ **DONE 2026-09-11** — **Order-service Kafka consumer for `payment.captured`/`payment_intent.payment_failed`**
   (§4.4) — the biggest real gap: without it, webhook-captured card orders can
   stay `PENDING_PAYMENT` forever. Mirrors the existing mail/notification
   consumer pattern exactly.
2. ✅ **DONE 2026-09-11** — **Release reservation on payment failure** (§4.5) — call `inventory.Release`
   when `Pay()` sees `Status=FAILED`, same call already used by `Cancel()`.
3. ✅ **DONE 2026-09-11** (highest-severity items) — **Tenant-scoping hardening** (§3) — added
   `tenant_id`/parent-id to order-service's order mutations + `order_returns.Resolve`,
   payment-service's `payments.Update`, product-service's `product_variants`/`product_images`,
   user-management's `memberships.UpdateRole/Delete`. Lower-severity remainder
   (permission grants, `carts.Touch`, `cart_items.SetQuantity`) still open — see §3.
4. ✅ **DONE 2026-09-11** — **Payment webhook concurrency fix** (§4.4) — wrapped `HandleWebhook` in
   `RunInTx` + `SELECT...FOR UPDATE`, matching the existing inventory pattern.
   Verified under 8 real concurrent deliveries — exactly one capture, one notification, one email.
5. ✅ **DONE 2026-09-11** — **Coupon/order atomicity** (§5, P1 coupon) — `RedeemCoupon` now runs inside the
   same transaction as the order INSERT; a failed redemption rolls the whole order back.
   Verified under a 6-way concurrent redemption race — exactly 1 of 6 succeeded, counter stayed at 1.
6. ✅ **DONE 2026-09-11** — **Return over-redemption guard** (§5, item 12) — checks cumulative
   previously-returned (and pending-REQUESTED) quantity per SKU, not just the
   original order line quantity. Rejected returns correctly free their quantity back up.
7. ✅ **DONE 2026-09-11** — **Order cancel → void pending Stripe PaymentIntent** (§4.6).
   Also closed the follow-on race: a delayed capture webhook arriving after
   cancellation no longer resurrects the order — it's recorded and
   auto-refunded instead (reusing the existing refund path). Verified under a
   forced hard-race (void itself rejected, capture still lands) as well as the
   common case. See `ApiTestResults/README.md` § "Order cancellation lifecycle".
8. **Product hard-delete guard** (§5, item 9) — block/require-force deleting a
   product that has order history, or switch the "delete" action to only ever
   archive.
9. ✅ **DONE 2026-09-11** — **Password reset / change-password flow** (§4.1).
   Added `password_reset_tokens` (single-use, hashed, short-lived, same
   pattern as `refresh_tokens`), `POST /auth/change-password`,
   `POST /auth/forgot-password`, `POST /auth/reset-password`, and a
   `RevokeAllForUser` primitive so both flows invalidate every session for
   the account, not just issue new tokens. Email delivery reuses
   mail-service's existing `/internal/mail` endpoint and template registry
   (same client shape notification-service already had) — no new mail
   infrastructure. Verified end-to-end for real, including the actual
   delivered email (via the dev-only Mailpit catcher already in
   `docker-compose.dev.yml`). See `ApiTestResults/README.md` §
   "Password / account security lifecycle".
10. **Idempotent-checkout release-leak fix** (§4.5) — make the compensating
    `inventory.Release` on a losing idempotency race retry-safe instead of
    fire-and-forget (or move both `Reserve`/loser-`Release` inside a pattern
    that can't silently drop the compensation).
11. ✅ **DONE 2026-09-11** — **Subscription renewal + ongoing enforcement**
    (§5, item 8). Decision made: kept the existing manual-periods model
    (renew-by-calling-`Subscribe`-again) rather than real Stripe recurring
    billing — the codebase's own `Subscribe` was already an intentional
    charge-stub. Closed the actual root-cause bug (a lapsed subscription's
    `status` never flipped off `ACTIVE`, so the partial unique index silently
    blocked all renewal forever); added lazy expiry, renewal/reactivation
    with an `Idempotency-Key`-guarded no-double-extend path, and a
    `PAST_DUE` grace state for a failed renewal charge (internal
    `mark-past-due` endpoint standing in for a future real
    `invoice.payment_failed` webhook). Ongoing enforcement decision: tenant
    entitlement stays a one-time creation gate only — an existing tenant's
    data/operations are never re-gated by the owner's subscription status,
    matching this codebase's existing separation of subscription (user-scoped)
    from tenant ownership/data. See `ApiTestResults/README.md` § "Subscription
    lifecycle".
12. Smaller items as time allows: per-user coupon limit, review
    purchase-eligibility (needs a design decision first), Kafka DLQ, order
    payment_status PARTIALLY_REFUNDED state, Stripe webhook timestamp-tolerance
    check.

Items 8 and the review-eligibility piece of item 9-adjacent (§5 item 16) need a
product/architecture decision from you before I implement — everything else I
can build directly against the existing patterns.

**Waiting for your review/prioritization before touching any code**, per your
instruction.
