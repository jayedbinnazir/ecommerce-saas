# user-management service

Handles platform users, stores (tenants), memberships, roles, permissions,
authentication (local + Google/Facebook), and subscription billing (stubbed).

- **Stack:** Go + Gin, PostgreSQL (`database/sql` + `lib/pq`), Redis, JWT cookies
- **Base URL:** `http://localhost:8080/api/v1`
- **Module path:** `github.com/jayedbinnazir/golang-saas.git`

---

## Running it

```bash
# 1. start postgres + redis (docker compose, or your own)
# 2. apply the schema
APP_ENV=dev go run ./cmd/migration up        # also: down [n] | status

# 3. run the API
APP_ENV=dev go run ./cmd/api
```

### Config (`configs/config.yaml`, override with env)

| Key | Env | Notes |
|---|---|---|
| `jwt.secret` | `USER_JWT_SECRET` | HMAC secret for access tokens |
| `jwt.access_ttl` | `USER_JWT_ACCESS_TTL` | e.g. `15m` |
| `jwt.refresh_ttl` | `USER_JWT_REFRESH_TTL` | e.g. `7d`, `2w`, `168h` |
| `auth.frontend_url` | `USER_AUTH_FRONTEND_URL` | OAuth redirect target, CORS origin |
| `auth.cookie_secure` | `USER_AUTH_COOKIE_SECURE` | `true` in production (HTTPS) |
| `auth.google.client_id` / `client_secret` / `redirect_url` | `USER_AUTH_GOOGLE_*` | from Google Cloud console |
| `auth.facebook.client_id` / `client_secret` / `redirect_url` | `USER_AUTH_FACEBOOK_*` | from Meta app dashboard |

---

## Conventions

### Success envelope

```json
{ "status": "success", "data": { } }
```

List endpoints that paginate (`?limit`, `?offset`) wrap the data:

```json
{ "status": "success", "data": { "items": [ ], "limit": 20, "offset": 0 } }
```

`204 No Content` responses have an empty body.

### Error envelope

Every error response is self-describing — it names the service and route, carries
a timestamp and the `request_id` (also on the `X-Request-Id` response header) so
you can grep the logs, and, for validation failures, lists exactly which fields
are wrong.

```json
{
  "status": "error",
  "error": {
    "service": "user-management",
    "code": "VALIDATION_ERROR",
    "message": "one or more fields are invalid",
    "method": "POST",
    "path": "/api/v1/auth/register",
    "request_id": "d61efc9c-131c-4769-87ed-d93bbd34ba5e",
    "timestamp": "2026-09-08T10:24:15Z",
    "fields": [
      { "field": "name",     "message": "this field is required" },
      { "field": "email",    "message": "must be a valid email address" },
      { "field": "password", "message": "must be at least 8 characters" }
    ]
  }
}
```

| field | always present | notes |
|---|---|---|
| `service` | yes | `user-management` |
| `code` | yes | machine-readable, see table below |
| `message` | yes | human summary |
| `method`, `path` | yes | the request that failed |
| `request_id` | yes* | matches the log line and `X-Request-Id` header |
| `timestamp` | yes | RFC 3339, UTC |
| `fields` | only on `VALIDATION_ERROR` from body binding | `[{field, message}]` using JSON field names |

\* absent only if the request somehow bypassed the `RequestID` middleware.

Non-validation errors have the same shape minus `fields`:

```json
{
  "status": "error",
  "error": {
    "service": "user-management",
    "code": "CONFLICT",
    "message": "user already exists",
    "method": "POST",
    "path": "/api/v1/auth/register",
    "request_id": "41496114-e45d-41f6-8f3f-515815c15f87",
    "timestamp": "2026-09-08T10:24:22Z"
  }
}
```

| `code` | HTTP | When |
|---|---|---|
| `BAD_REQUEST` | 400 | malformed path param, non-JSON body, missing query/redirect param |
| `UNAUTHORIZED` | 401 | missing/invalid/expired session, wrong credentials |
| `FORBIDDEN` | 403 | authenticated but not allowed (role/ownership) |
| `NOT_FOUND` | 404 | resource does not exist |
| `CONFLICT` | 409 | duplicate email / slug / membership |
| `VALIDATION_ERROR` | 422 | body failed field validation or a domain rule |
| `RATE_LIMITED` | 429 | too many requests from one IP |
| `INTERNAL_ERROR` | 500 | unexpected failure (details in the service log, keyed by `request_id`) |

> The per-endpoint examples below abbreviate the error body to `code` + `message`;
> the real response always includes the envelope fields shown above.

### Authentication

On login the API sets two `httpOnly` cookies and also returns the tokens in the body:

- `access_token` — short-lived JWT, sent on every request (`path=/`)
- `refresh_token` — opaque, sent only to `/api/v1/auth/*` (`path=/api/v1/auth`)

Browsers just send the cookies. Non-browser clients can send
`Authorization: Bearer <access_token>` instead.

### Roles

| Role | Scope | Meaning |
|---|---|---|
| `SUPER_ADMIN` | platform | operator; `users.is_super_admin = true` (set in DB, never via API) |
| `ADMIN` | tenant | store owner; created automatically when a store is registered |
| `MANAGER` | tenant | staff added by an admin |
| `CUSTOMER` | tenant | shopper |

---

## Endpoints

Legend: 🔓 public · 🔑 any logged-in user · 👑 super-admin · 🏪 tenant ADMIN · 🏪👔 tenant ADMIN or MANAGER

| Method | Path | Access |
|---|---|---|
| GET | `/health` | 🔓 |
| POST | `/auth/register` | 🔓 |
| POST | `/auth/login` | 🔓 |
| POST | `/auth/refresh` | 🔓 (needs refresh cookie) |
| POST | `/auth/logout` | 🔑 |
| GET | `/auth/me` | 🔑 |
| GET | `/auth/{provider}/login` | 🔓 |
| GET | `/auth/{provider}/callback` | 🔓 |
| GET | `/billing/plans` | 🔓 |
| GET | `/billing/subscription` | 🔑 |
| POST | `/billing/subscription` | 🔑 |
| DELETE | `/billing/subscription` | 🔑 |
| POST | `/tenants` | 🔑 (needs active subscription) |
| GET | `/tenants` | 🔑 (your stores) |
| GET | `/tenants/{tenantId}` | 🏪 owner or 👑 |
| PATCH | `/tenants/{tenantId}` | 🏪 owner or 👑 |
| DELETE | `/tenants/{tenantId}` | 🏪 owner or 👑 |
| GET | `/admin/tenants` | 👑 |
| GET | `/tenants/{tenantId}/members` | 🏪👔 |
| POST | `/tenants/{tenantId}/members` | 🏪 |
| PATCH | `/tenants/{tenantId}/members/{userId}` | 🏪 |
| DELETE | `/tenants/{tenantId}/members/{userId}` | 🏪 |
| GET | `/tenants/{tenantId}/members/{userId}/permissions` | 🏪 |
| POST | `/tenants/{tenantId}/members/{userId}/permissions` | 🏪 |
| DELETE | `/tenants/{tenantId}/members/{userId}/permissions/{permissionId}` | 🏪 |
| GET | `/me/memberships` | 🔑 |
| GET | `/users/{userId}/addresses` | 🔑 self or 👑 |
| POST | `/users/{userId}/addresses` | 🔑 self or 👑 |
| GET | `/users/{userId}/addresses/{addressId}` | 🔑 self or 👑 |
| PATCH | `/users/{userId}/addresses/{addressId}` | 🔑 self or 👑 |
| DELETE | `/users/{userId}/addresses/{addressId}` | 🔑 self or 👑 |
| GET / GET / PATCH / DELETE | `/users`, `/users/{userId}` | 👑 |
| POST / GET / GET / PATCH / DELETE | `/roles`, `/roles/{roleId}` | 👑 |
| POST / GET / GET / PATCH / DELETE | `/permissions`, `/permissions/{id}` | 👑 |
| GET / POST / DELETE | `/roles/{roleId}/permissions[/{permissionId}]` | 👑 |

---

## Typical frontend flow

1. `POST /auth/register` or `POST /auth/login` → cookies set.
2. `GET /billing/plans` → show pricing.
3. `POST /billing/subscription` `{ "plan_code": "starter-monthly" }` → subscription active.
4. `POST /tenants` `{ "name": "...", "slug": "..." }` → store created, caller becomes its `ADMIN`.
5. `POST /tenants/{id}/members` → add managers, then grant them permissions.

---

## Auth

### `POST /auth/register`

```json
{ "name": "Ada Lovelace", "email": "ada@example.com", "password": "s3cret-passphrase" }
```

**201 Created** (and `Set-Cookie: access_token=...`, `refresh_token=...`)

```json
{
  "status": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "K8s3f...redacted",
    "expires_in": 900
  }
}
```

**409 Conflict** — email already registered

```json
{ "status": "error", "error": { "code": "CONFLICT", "message": "user already exists" } }
```

**422 Unprocessable Entity** — bad body

```json
{ "status": "error", "error": { "code": "VALIDATION_ERROR", "message": "invalid request body" } }
```

### `POST /auth/login`

```json
{ "email": "ada@example.com", "password": "s3cret-passphrase" }
```

**200 OK** — same shape as register.

**401 Unauthorized** — wrong email or password (identical message either way)

```json
{ "status": "error", "error": { "code": "UNAUTHORIZED", "message": "invalid credentials" } }
```

### `POST /auth/refresh`

No body — sends the `refresh_token` cookie. Rotates it: the old refresh token is
revoked, a new access + refresh pair is issued. Replaying a revoked token kills the
whole token family.

**200 OK** — same shape as login.

**401 Unauthorized**

```json
{ "status": "error", "error": { "code": "UNAUTHORIZED", "message": "refresh token is invalid or expired" } }
```

### `POST /auth/logout` 🔑

No body. Revokes the session's refresh tokens and clears the cookies. → **204 No Content**

### `GET /auth/me` 🔑

**200 OK**

```json
{
  "status": "success",
  "data": {
    "id": "0b6b8f2e-1c1a-4f9e-9d2a-2a1b3c4d5e6f",
    "name": "Ada Lovelace",
    "email": "ada@example.com",
    "is_super_admin": false,
    "created_at": "2026-09-08T09:20:01Z",
    "updated_at": "2026-09-08T09:20:01Z"
  }
}
```

### `GET /auth/{provider}/login` — `provider` is `google` or `facebook`

Optional `?redirect=<url>` — where to send the browser after a successful callback
(defaults to `auth.frontend_url`). Responds **302 Found** to the provider consent screen.

**400 Bad Request** — unknown provider

```json
{ "status": "error", "error": { "code": "BAD_REQUEST", "message": "unknown provider" } }
```

### `GET /auth/{provider}/callback?code=...&state=...`

The provider redirects here. The service exchanges the code, finds or creates the
user (linking by verified email), issues a session, sets cookies, and responds
**302 Found** back to the frontend.

**401 Unauthorized** — provider returned no verified email

```json
{ "status": "error", "error": { "code": "UNAUTHORIZED", "message": "the provider did not return a verified email" } }
```

---

## Billing

### `GET /billing/plans` 🔓

**200 OK**

```json
{
  "status": "success",
  "data": [
    { "id": "…", "code": "starter-monthly", "name": "Starter (Monthly)", "interval": "MONTH", "price_cents": 2900, "currency": "USD" },
    { "id": "…", "code": "starter-yearly",  "name": "Starter (Yearly)",  "interval": "YEAR",  "price_cents": 29000, "currency": "USD" }
  ]
}
```

### `POST /billing/subscription` 🔑

Payment is stubbed — this just records an ACTIVE subscription for one interval.

```json
{ "plan_code": "starter-monthly" }
```

**201 Created**

```json
{
  "status": "success",
  "data": {
    "id": "…",
    "plan_id": "…",
    "status": "ACTIVE",
    "current_period_start": "2026-09-08T09:25:00Z",
    "current_period_end": "2026-10-08T09:25:00Z"
  }
}
```

**409 Conflict** — already has an active subscription

```json
{ "status": "error", "error": { "code": "CONFLICT", "message": "user already has an active subscription" } }
```

**404 Not Found** — unknown `plan_code`

```json
{ "status": "error", "error": { "code": "NOT_FOUND", "message": "plan not found" } }
```

### `GET /billing/subscription` 🔑

**200 OK** — same shape as above.

**404 Not Found**

```json
{ "status": "error", "error": { "code": "NOT_FOUND", "message": "no active subscription" } }
```

### `DELETE /billing/subscription` 🔑

Cancels at period end (`status` becomes `CANCELED`, access continues until
`current_period_end`). **200 OK** with the updated subscription.

---

## Tenants (stores)

### `POST /tenants` 🔑

Requires an active subscription. The caller becomes the store's `ADMIN`.

```json
{ "name": "Ada's Gadgets", "slug": "adas-gadgets" }
```

`slug` must be lowercase letters/digits separated by single hyphens.

**201 Created**

```json
{
  "status": "success",
  "data": {
    "id": "3f7c1e90-2b6d-4a11-8f0e-9c2d1a4b5e6f",
    "name": "Ada's Gadgets",
    "slug": "adas-gadgets",
    "status": "ACTIVE",
    "owner_user_id": "0b6b8f2e-1c1a-4f9e-9d2a-2a1b3c4d5e6f",
    "created_at": "2026-09-08T09:30:00Z",
    "updated_at": "2026-09-08T09:30:00Z"
  }
}
```

**422 Unprocessable Entity** — no subscription

```json
{ "status": "error", "error": { "code": "VALIDATION_ERROR", "message": "an active subscription is required to create a store" } }
```

**409 Conflict** — slug taken

```json
{ "status": "error", "error": { "code": "CONFLICT", "message": "tenant slug already taken" } }
```

### `GET /tenants` 🔑

Stores the caller owns. **200 OK**

```json
{ "status": "success", "data": [ { "id": "…", "name": "…", "slug": "…", "status": "ACTIVE", "owner_user_id": "…", "created_at": "…", "updated_at": "…" } ] }
```

### `GET /tenants/{tenantId}` · `PATCH /tenants/{tenantId}` · `DELETE /tenants/{tenantId}`

Owner or super-admin only.

`PATCH` body (both fields optional):

```json
{ "name": "Ada's Gadget Emporium", "status": "SUSPENDED" }
```

`status` ∈ `ACTIVE | SUSPENDED | INACTIVE`. → **200 OK** with the tenant. `DELETE` → **204**.

**403 Forbidden** — not the owner

```json
{ "status": "error", "error": { "code": "FORBIDDEN", "message": "not allowed to manage this tenant" } }
```

### `GET /admin/tenants?limit=20&offset=0` 👑

**200 OK**

```json
{ "status": "success", "data": { "items": [ /* tenants */ ], "limit": 20, "offset": 0 } }
```

---

## Memberships

### `GET /tenants/{tenantId}/members` 🏪👔

**200 OK**

```json
{
  "status": "success",
  "data": [
    {
      "id": "…", "tenant_id": "…", "user_id": "…", "role_id": "…",
      "role_name": "ADMIN",
      "created_at": "…", "updated_at": "…"
    }
  ]
}
```

### `POST /tenants/{tenantId}/members` 🏪

```json
{ "user_id": "9a1b2c3d-4e5f-6071-8293-a4b5c6d7e8f9", "role": "MANAGER" }
```

`role` ∈ `MANAGER | CUSTOMER` (admins cannot be assigned here).

**201 Created**

```json
{
  "status": "success",
  "data": { "id": "…", "tenant_id": "…", "user_id": "…", "role_id": "…", "created_at": "…", "updated_at": "…" }
}
```

**409 Conflict**

```json
{ "status": "error", "error": { "code": "CONFLICT", "message": "user is already a member of this tenant" } }
```

**422 Unprocessable Entity** — referenced user/role missing

```json
{ "status": "error", "error": { "code": "VALIDATION_ERROR", "message": "referenced tenant, user or role does not exist" } }
```

### `PATCH /tenants/{tenantId}/members/{userId}` 🏪

```json
{ "role": "CUSTOMER" }
```

→ **200 OK** with the membership.

**422 Unprocessable Entity** — trying to change the owner

```json
{ "status": "error", "error": { "code": "VALIDATION_ERROR", "message": "the tenant owner's membership cannot be changed here" } }
```

### `DELETE /tenants/{tenantId}/members/{userId}` 🏪

→ **204 No Content**.

### Per-member permissions 🏪

A member's effective permissions = their role's permissions **plus** any granted
individually here.

`GET /tenants/{tenantId}/members/{userId}/permissions` → **200 OK**

```json
{ "status": "success", "data": { "permissions": ["order:read", "product:read", "product:write"] } }
```

`POST /tenants/{tenantId}/members/{userId}/permissions`

```json
{ "permission_id": "c1d2e3f4-a5b6-7890-c1d2-e3f4a5b60718" }
```

→ **204 No Content**. `DELETE .../permissions/{permissionId}` → **204**.

### `GET /me/memberships` 🔑

Every tenant the caller belongs to. **200 OK** — array of memberships (with `role_name`).

---

## Addresses

Path `/users/{userId}/addresses` — the caller must be that user, or a super-admin.

### `POST /users/{userId}/addresses`

```json
{
  "label": "Home",
  "recipient_name": "Ada Lovelace",
  "phone": "+15551234567",
  "address_line_1": "12 Analytical Engine Way",
  "address_line_2": "Apt 4",
  "city": "London",
  "state": null,
  "postal_code": "EC1A 1BB",
  "country": "GB",
  "is_default_shipping": true,
  "is_default_billing": false
}
```

`country` must be a 2-letter uppercase ISO code. Setting a default clears the
previous default of that kind for the user (one default shipping + one default
billing max).

**201 Created**

```json
{
  "status": "success",
  "data": {
    "id": "…", "user_id": "…",
    "label": "Home", "recipient_name": "Ada Lovelace", "phone": "+15551234567",
    "address_line_1": "12 Analytical Engine Way", "address_line_2": "Apt 4",
    "city": "London", "state": null, "postal_code": "EC1A 1BB", "country": "GB",
    "is_default_shipping": true, "is_default_billing": false,
    "created_at": "…", "updated_at": "…"
  }
}
```

### `GET /users/{userId}/addresses`

**200 OK** — `{ "status": "success", "data": [ /* addresses */ ] }`

### `GET|PATCH|DELETE /users/{userId}/addresses/{addressId}`

`PATCH` — any subset of the create fields. → **200 OK** with the address. `DELETE` → **204**.

**403 Forbidden** — someone else's address

```json
{ "status": "error", "error": { "code": "FORBIDDEN", "message": "you can only manage your own addresses" } }
```

---

## Users (super-admin)

### `GET /users?limit=20&offset=0` 👑

**200 OK** — `{ "status": "success", "data": { "items": [ /* users */ ], "limit": 20, "offset": 0 } }`

### `GET /users/{userId}` 👑 → **200 OK** with a user object.

### `PATCH /users/{userId}` 👑

```json
{ "name": "Ada King", "email": "ada.king@example.com" }
```

Both optional. Password changes are not done here. → **200 OK** with the user.

### `DELETE /users/{userId}` 👑 → **204 No Content**.

**403 Forbidden** — not a super-admin

```json
{ "status": "error", "error": { "code": "FORBIDDEN", "message": "super-admin access required" } }
```

---

## Roles (super-admin)

### `POST /roles` 👑

```json
{ "name": "MANAGER", "description": "Store staff" }
```

`name` ∈ `SUPER_ADMIN | ADMIN | MANAGER | CUSTOMER`. The four roles are seeded by
migration, so this mostly matters for re-creating a deleted one.

**201 Created**

```json
{
  "status": "success",
  "data": { "id": "…", "name": "MANAGER", "description": "Store staff", "created_at": "…", "updated_at": "…" }
}
```

**409 Conflict** — `{ "code": "CONFLICT", "message": "role already exists" }`

### `GET /roles?limit=20&offset=0` 👑 — list envelope.
### `GET|PATCH|DELETE /roles/{roleId}` 👑

`PATCH` body: `{ "name": "...", "description": "..." }` (both optional). `DELETE` → **204**.

**422 Unprocessable Entity** — role still assigned to members

```json
{ "status": "error", "error": { "code": "VALIDATION_ERROR", "message": "role is still assigned to one or more members" } }
```

---

## Permissions (super-admin)

### `POST /permissions` 👑

```json
{ "name": "invoice:write", "description": "Create and edit invoices" }
```

**201 Created**

```json
{
  "status": "success",
  "data": { "id": "…", "name": "invoice:write", "description": "Create and edit invoices", "created_at": "…", "updated_at": "…" }
}
```

### `GET /permissions?limit=20&offset=0` 👑 — list envelope.
### `GET|PATCH|DELETE /permissions/{id}` 👑 — standard CRUD, `DELETE` → **204**.

### Role ↔ permission grants 👑

`GET /roles/{roleId}/permissions` → **200 OK**

```json
{ "status": "success", "data": [ { "id": "…", "name": "product:read", "description": "View products", "created_at": "…", "updated_at": "…" } ] }
```

`POST /roles/{roleId}/permissions`

```json
{ "permission_id": "c1d2e3f4-a5b6-7890-c1d2-e3f4a5b60718" }
```

→ **204 No Content**. `DELETE /roles/{roleId}/permissions/{permissionId}` → **204**.

**404 Not Found** — grant does not exist (on delete)

```json
{ "status": "error", "error": { "code": "NOT_FOUND", "message": "grant not found" } }
```

---

## curl walkthrough

```bash
BASE=http://localhost:8080/api/v1
JAR=cookies.txt

# register (stores cookies in the jar)
curl -s -c $JAR -X POST $BASE/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"Ada","email":"ada@example.com","password":"s3cret-passphrase"}'

# who am I
curl -s -b $JAR $BASE/auth/me

# subscribe, then create a store
curl -s -b $JAR -X POST $BASE/billing/subscription \
  -H 'Content-Type: application/json' -d '{"plan_code":"starter-monthly"}'

curl -s -b $JAR -X POST $BASE/tenants \
  -H 'Content-Type: application/json' \
  -d '{"name":"Ada'\''s Gadgets","slug":"adas-gadgets"}'

# add a manager to that store (TENANT_ID / USER_ID from earlier responses)
curl -s -b $JAR -X POST $BASE/tenants/$TENANT_ID/members \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"'"$USER_ID"'","role":"MANAGER"}'

# refresh the session, then log out
curl -s -b $JAR -c $JAR -X POST $BASE/auth/refresh
curl -s -b $JAR -X POST $BASE/auth/logout
```


this service hv a problem to push in github
