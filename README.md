# SSO Server

A lightweight OAuth2 / OpenID Connect (OIDC) Single Sign-On server written in Go, backed by SQLite.

## Features

- Authorization Code flow with PKCE (`S256` and `plain`)
- OpenID Connect ID tokens
- Refresh tokens
- RS256 JWT signing (auto-generated RSA key pair)
- User roles: `manager` / `staff`
- Admin API protected by a shared secret
- Zero external dependencies at runtime (SQLite embedded via CGo-free driver)

---

## Quick Start

```bash
# Copy and edit environment config
cp .env.example .env   # or edit .env directly

# Run
go run .
```

The server starts on `:8080` by default. The RSA private key is generated and saved to `private.pem` on first run.

---

## Configuration

All settings are read from environment variables (`.env` is loaded by your shell or a tool like `direnv`).

| Variable        | Default                    | Description                              |
|-----------------|----------------------------|------------------------------------------|
| `SERVER_ADDR`   | `:8080`                    | TCP address the server listens on        |
| `ISSUER`        | `http://localhost:8080`    | OIDC issuer URL (must be publicly reachable in prod) |
| `DATABASE_PATH` | `sso.db`                   | Path to the SQLite database file         |
| `ADMIN_SECRET`  | `change-me-in-production`  | Secret for the Admin API (`X-Admin-Secret` header) |

> **Production note:** Change `ADMIN_SECRET` and set `ISSUER` to your real HTTPS domain.

---

## Endpoints

### OIDC Discovery

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/.well-known/openid-configuration` | OIDC discovery document |
| `GET` | `/.well-known/jwks.json` | JSON Web Key Set (public key for verifying JWTs) |

### Authorization

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/authorize` | Displays the login UI for a client |
| `POST` | `/authorize` | Authenticates the user and issues an authorization code |

**Query parameters for `GET /authorize`:**

| Parameter | Required | Description |
|-----------|----------|-------------|
| `client_id` | Yes | Registered client ID |
| `redirect_uri` | Yes | Must match one of the client's registered redirect URIs |
| `response_type` | Yes | Must be `code` |
| `scope` | No | Space-separated scopes (`openid`, `profile`, `email`) |
| `state` | No | Opaque value for CSRF protection |
| `nonce` | No | Nonce included in the ID token |
| `code_challenge` | No | PKCE challenge value |
| `code_challenge_method` | No | `S256` or `plain` |

### Token

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/token` | Exchange a code or refresh token for access/ID tokens |

**Client authentication:** HTTP Basic Auth (`Authorization: Basic ...`) or form fields (`client_id` + `client_secret`).

#### Grant: `authorization_code`

| Parameter | Required | Description |
|-----------|----------|-------------|
| `grant_type` | Yes | `authorization_code` |
| `code` | Yes | Authorization code from `/authorize` |
| `redirect_uri` | Yes | Must match the one used in the authorize request |
| `code_verifier` | If PKCE used | PKCE code verifier |

#### Grant: `refresh_token`

| Parameter | Required | Description |
|-----------|----------|-------------|
| `grant_type` | Yes | `refresh_token` |
| `refresh_token` | Yes | Refresh token from a previous token response |

**Token response:**

```json
{
  "access_token": "<JWT>",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "<opaque>",
  "scope": "openid profile email",
  "id_token": "<JWT>"
}
```

Token lifetimes:
- Authorization codes: **10 minutes** (single-use)
- Access tokens: **1 hour**
- Refresh tokens: **30 days** (single-use; a new one is issued on each refresh)

### Userinfo

| Method | Path | Description |
|--------|------|-------------|
| `GET` / `POST` | `/userinfo` | Returns claims about the authenticated user |

**Request header:** `Authorization: Bearer <access_token>`

**Response** (claims depend on granted scopes):

```json
{
  "sub": "<user-id>",
  "email": "user@example.com",
  "name": "Alice",
  "role": "manager"
}
```

| Claim | Scope required |
|-------|---------------|
| `sub` | always present |
| `email` | `email` |
| `name`, `role` | `profile` |

### User Registration

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/register` | Self-service user registration |

**Request body:**

```json
{
  "email": "user@example.com",
  "password": "min8chars",
  "name": "Alice",
  "role": "staff"
}
```

- `role` is optional; defaults to `staff` if omitted or invalid.
- Valid roles: `manager`, `staff`.
- Returns `409 Conflict` if the email is already registered.

---

## Admin API

All admin endpoints require the `X-Admin-Secret` header matching the `ADMIN_SECRET` environment variable.

### Clients

#### Create client

```
POST /admin/clients
X-Admin-Secret: <secret>
Content-Type: application/json
```

```json
{
  "client_id": "my-app",
  "client_secret": "s3cr3t",
  "name": "My Application",
  "redirect_uris": ["https://myapp.example.com/callback"],
  "scopes": "openid profile email"
}
```

- `scopes` defaults to `openid profile email` if omitted.
- Returns `201 Created` with the created client (secret is never returned).

#### List clients

```
GET /admin/clients
X-Admin-Secret: <secret>
```

Returns an array of client objects (no secrets).

### Users

#### Create user

```
POST /admin/users
X-Admin-Secret: <secret>
Content-Type: application/json
```

```json
{
  "email": "alice@example.com",
  "password": "min8chars",
  "name": "Alice",
  "role": "manager"
}
```

#### List users

```
GET /admin/users
X-Admin-Secret: <secret>
```

#### Update user role

```
PATCH /admin/users/{id}/role
X-Admin-Secret: <secret>
Content-Type: application/json
```

```json
{ "role": "manager" }
```

Valid roles: `manager`, `staff`.

---

## JWT Token Structure

Tokens are signed with RS256 using the key in `private.pem`.

**Access token claims:**

| Claim | Value |
|-------|-------|
| `iss` | Issuer URL |
| `sub` | User ID |
| `aud` | `[client_id]` |
| `iat` | Issued at (Unix) |
| `exp` | Expiry (Unix, +1h) |
| `scope` | Granted scopes |
| `role` | User role |

**ID token claims** (when `openid` scope is granted):

| Claim | Scope |
|-------|-------|
| `iss`, `sub`, `aud`, `iat`, `exp` | always |
| `nonce` | if provided in the authorize request |
| `email` | `email` scope |
| `name`, `role` | `profile` scope |

---

## Database Schema

SQLite database with four tables:

- **`users`** — `id`, `email`, `password_hash` (bcrypt), `name`, `role`, `created_at`
- **`clients`** — `id`, `client_id`, `client_secret_hash` (bcrypt), `name`, `redirect_uris` (JSON), `scopes`, `created_at`
- **`authorization_codes`** — single-use, 10-minute TTL, supports PKCE
- **`refresh_tokens`** — single-use (rotated on use), 30-day TTL

---

## Integration Example

### 1. Register a client (admin)

```bash
curl -X POST http://localhost:8080/admin/clients \
  -H "X-Admin-Secret: change-me-in-production" \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "my-app",
    "client_secret": "supersecret",
    "name": "My App",
    "redirect_uris": ["http://localhost:3000/callback"]
  }'
```

### 2. Create a user (admin)

```bash
curl -X POST http://localhost:8080/admin/users \
  -H "X-Admin-Secret: change-me-in-production" \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"password123","name":"Alice","role":"staff"}'
```

### 3. Start the Authorization Code flow

Redirect the user's browser to:

```
http://localhost:8080/authorize
  ?client_id=my-app
  &redirect_uri=http://localhost:3000/callback
  &response_type=code
  &scope=openid%20profile%20email
  &state=random-csrf-token
```

### 4. Exchange the code for tokens

```bash
curl -X POST http://localhost:8080/token \
  -u "my-app:supersecret" \
  -d "grant_type=authorization_code" \
  -d "code=<code>" \
  -d "redirect_uri=http://localhost:3000/callback"
```

### 5. Call Userinfo

```bash
curl http://localhost:8080/userinfo \
  -H "Authorization: Bearer <access_token>"
```

---

## Error Responses

OAuth2 errors follow [RFC 6749](https://datatracker.ietf.org/doc/html/rfc6749):

```json
{
  "error": "invalid_grant",
  "error_description": "authorization code expired"
}
```

Common error codes: `invalid_request`, `invalid_client`, `invalid_grant`, `unsupported_grant_type`, `server_error`, `unauthorized`, `conflict`, `not_found`.