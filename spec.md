# Spec: Mini BigC SSO

## Objective

Build and maintain the Mini BigC SSO service as the local OAuth2/OpenID Connect identity provider for Mini BigC applications.

The SSO service must provide:

- OpenID Provider discovery metadata.
- JWKS endpoint for RS256 token verification.
- Authorization Code flow with login page.
- Optional PKCE verification for authorization code exchange.
- Access tokens, ID tokens, and refresh tokens.
- Userinfo endpoint.
- User self-service registration.
- Admin APIs for clients and users.
- SQLite-backed local persistence.
- RSA private key load/generation for JWT signing.

Success means the Mini BigC frontend can redirect users through the SSO service, receive OIDC tokens, retrieve user profile claims, and map SSO roles into the app role model.

## Assumptions

- SSO package is `miniBigCProject_SSO`.
- Module name is `sso`.
- Runtime port defaults to `8080`.
- Issuer defaults to `http://localhost:8080`.
- Database defaults to local SQLite file `sso.db`.
- RSA signing key defaults to `private.pem`.
- Supported roles are `manager` and `staff`.
- Supported OAuth response type is `code`.
- Supported token grants are `authorization_code` and `refresh_token`.
- Supported signing algorithm is `RS256`.
- The service is currently intended for local/demo/internal development, not hardened production identity management.

## Tech Stack

- Language: Go 1.25.0 as declared in `go.mod`.
- HTTP server: Go `net/http` with `http.NewServeMux`.
- Database: SQLite via `modernc.org/sqlite`.
- Password/client secret hashing: `golang.org/x/crypto/bcrypt`.
- JWT: `github.com/golang-jwt/jwt/v5`.
- Token and ID generation: `crypto/rand` with base64url encoding.
- RSA keys: Go `crypto/rsa`, `crypto/x509`, PEM files.

## Commands

Run commands from `miniBigCProject_SSO`.

```bash
go run .
go build ./...
go test ./...
go vet ./...
```

Local URLs:

```text
Issuer:     http://localhost:8080
Discovery:  http://localhost:8080/.well-known/openid-configuration
JWKS:       http://localhost:8080/.well-known/jwks.json
Authorize:  http://localhost:8080/authorize
Token:      http://localhost:8080/token
Userinfo:   http://localhost:8080/userinfo
Register:   http://localhost:8080/register
```

## Runtime Configuration

Configuration is loaded from environment variables in `internal/config/config.go`.

| Variable | Default | Purpose |
| --- | --- | --- |
| `SERVER_ADDR` | `:8080` | HTTP listen address |
| `ISSUER` | `http://localhost:8080` | OIDC issuer and endpoint base URL |
| `DATABASE_PATH` | `sso.db` | SQLite database path |
| `ADMIN_SECRET` | `change-me-in-production` | Admin API shared secret |

Requirements:

- `ISSUER` must match the public URL clients use for discovery and token validation.
- `ADMIN_SECRET` must be overridden outside local/demo environments.
- `DATABASE_PATH` must be writable by the running process.
- `private.pem` must be protected with file mode `0600` when generated.

## Project Structure

```text
miniBigCProject_SSO/
├── main.go                         # Server wiring and route registration
├── go.mod                          # Module and dependency versions
├── sso.db                          # Local SQLite database file
├── server.log                      # Local runtime log output
├── server.err                      # Local runtime error output
└── internal/
    ├── config/
    │   └── config.go               # Env config
    ├── db/
    │   └── db.go                   # SQLite schema and data access
    ├── handlers/
    │   ├── handlers.go             # Shared handler helpers
    │   ├── discovery.go            # OIDC discovery and JWKS
    │   ├── authorize.go            # Login page and auth code creation
    │   ├── token.go                # Token endpoint and JWT issuance
    │   ├── userinfo.go             # Userinfo endpoint
    │   ├── register.go             # Self-service user registration
    │   └── admin.go                # Admin client/user APIs
    └── keys/
        └── keys.go                 # RSA key load/generation
```

## Architecture

Current request flow:

```text
HTTP request
→ net/http ServeMux
→ handler method
→ db.DB helper
→ SQLite
```

Responsibilities:

- `main.go`: load config, open DB, load/generate RSA key, create handler, register routes, serve static assets, start server.
- `internal/config`: read environment variables and defaults.
- `internal/db`: migrate schema, create/read/update users, clients, authorization codes, and refresh tokens.
- `internal/handlers`: implement OIDC/OAuth/admin HTTP behavior and response formatting.
- `internal/keys`: load existing RSA private key or generate and persist a new one.

Boundaries:

- Handlers parse/validate HTTP input and choose status codes.
- DB helpers own SQL and transactional token/code usage.
- Key helpers own filesystem key handling.
- Passwords and client secrets are never stored in plaintext.

## Route Surface

### Discovery And Keys

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/.well-known/openid-configuration` | OIDC provider metadata |
| `GET` | `/.well-known/jwks.json` | Public JWKS for token verification |

### OAuth2/OIDC Core

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/authorize` | Render login form for authorization request |
| `POST` | `/authorize` | Validate credentials and redirect with authorization code |
| `POST` | `/token` | Exchange authorization code or refresh token for tokens |
| `GET` | `/userinfo` | Return claims for bearer access token |
| `POST` | `/userinfo` | Same as GET userinfo |

### Self-Service And Admin

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/register` | Create user account |
| `POST` | `/admin/clients` | Create OAuth client |
| `GET` | `/admin/clients` | List OAuth clients |
| `GET` | `/admin/users` | List users |
| `POST` | `/admin/users` | Create user |
| `PATCH` | `/admin/users/{id}/role` | Update user role |

### Static Assets

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/assets/*` | Serve frontend image assets when found |
| `GET` | `/assets/brand-logo.ico` | Serve Mini BigC icon when found |

## OIDC Discovery Contract

Endpoint:

```http
GET /.well-known/openid-configuration
```

Required response fields:

| Field | Value |
| --- | --- |
| `issuer` | configured issuer |
| `authorization_endpoint` | `{issuer}/authorize` |
| `token_endpoint` | `{issuer}/token` |
| `userinfo_endpoint` | `{issuer}/userinfo` |
| `jwks_uri` | `{issuer}/.well-known/jwks.json` |
| `registration_endpoint` | `{issuer}/register` |
| `response_types_supported` | `["code"]` |
| `subject_types_supported` | `["public"]` |
| `id_token_signing_alg_values_supported` | `["RS256"]` |
| `scopes_supported` | `["openid", "profile", "email"]` |
| `token_endpoint_auth_methods_supported` | `["client_secret_post", "client_secret_basic"]` |
| `claims_supported` | `["sub", "email", "name", "iat", "exp", "iss", "aud", "nonce"]` |
| `grant_types_supported` | `["authorization_code", "refresh_token"]` |
| `code_challenge_methods_supported` | `["S256", "plain"]` |

Acceptance criteria:

- Response content type is JSON.
- All endpoint URLs use configured `ISSUER`.
- Discovery should not require authentication.

## JWKS Contract

Endpoint:

```http
GET /.well-known/jwks.json
```

Response shape:

```json
{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "alg": "RS256",
      "kid": "key-1",
      "n": "...",
      "e": "..."
    }
  ]
}
```

Requirements:

- JWKS exposes only public RSA key material.
- `kid` must match token header `kid`.
- `alg` must be `RS256`.
- Endpoint must not expose private key material.

## Authorization Endpoint

### `GET /authorize`

Purpose:

- Validate client and redirect URI.
- Render Mini BigC SSO login page.
- Preserve OAuth parameters in hidden form fields.

Required query parameters:

| Parameter | Required | Rule |
| --- | --- | --- |
| `client_id` | yes | Must match registered client |
| `redirect_uri` | yes | Must exactly match registered URI |
| `response_type` | yes | Must be `code` |

Optional query parameters:

- `scope`
- `state`
- `nonce`
- `code_challenge`
- `code_challenge_method`

Failure cases:

| Case | Status | Body |
| --- | --- | --- |
| Missing required parameter | `400` | Plain text HTTP error |
| Unsupported response type | `400` | Plain text HTTP error |
| Unknown client | `400` | Plain text HTTP error |
| Redirect URI not registered | `400` | Plain text HTTP error |
| DB error | `500` | Plain text HTTP error |

Acceptance criteria:

- Login form uses `method="POST"` and `action="/authorize"`.
- Hidden fields preserve authorization request values.
- Password field supports show/hide with accessible button labels.
- Login page works on mobile and desktop.
- No credentials are placed in query string.

### `POST /authorize`

Purpose:

- Authenticate user credentials.
- Intersect requested scopes with client-allowed scopes.
- Ensure `openid` is included.
- Create one-time authorization code.
- Redirect back to `redirect_uri` with `code` and optional `state`.

Required form fields:

| Field | Required | Rule |
| --- | --- | --- |
| `client_id` | yes | Registered client |
| `redirect_uri` | yes | Exact match against registered URI |
| `email` | yes | Existing user email |
| `password` | yes | bcrypt password match |

Preserved form fields:

- `scope`
- `state`
- `nonce`
- `code_challenge`
- `code_challenge_method`

Success:

```text
302 Location: {redirect_uri}?code={code}&state={state}
```

Failure cases:

| Case | Status | Behavior |
| --- | --- | --- |
| Bad form parse | `400` | Plain text error |
| Invalid client/redirect | `400` | Plain text error |
| Unknown user | `401` | Render login form with generic error |
| Wrong password | `401` | Render login form with generic error |
| DB error | `401` | Render login form with generic internal error |
| Auth code create failure | `401` | Render login form with generic internal error |

Security requirements:

- Login failure must not reveal whether email or password was wrong.
- Redirect URI matching must be exact.
- State must be returned unchanged when present.
- Authorization code must expire after 10 minutes.
- Authorization code must be single-use.

## Token Endpoint

Endpoint:

```http
POST /token
```

Content type:

```text
application/x-www-form-urlencoded
```

Cache headers:

```text
Cache-Control: no-store
Pragma: no-cache
```

Client authentication methods:

- HTTP Basic auth.
- `client_id` and `client_secret` form fields.

Failure cases before grant handling:

| Case | Status | Error |
| --- | --- | --- |
| Cannot parse form | `400` | `invalid_request` |
| Missing client credentials | `401` | `invalid_client` |
| Unknown client | `401` | `invalid_client` |
| Wrong client secret | `401` | `invalid_client` |
| DB client lookup error | `500` | `server_error` |
| Unsupported grant | `400` | `unsupported_grant_type` |

### Authorization Code Grant

Required form fields:

| Field | Required | Rule |
| --- | --- | --- |
| `grant_type` | yes | `authorization_code` |
| `code` | yes | Existing unused authorization code |
| `redirect_uri` | yes | Must match code's redirect URI |
| `code_verifier` | conditional | Required if code has challenge |

Validation:

- Code must exist.
- Code must not be used.
- Code must not be expired.
- Code must belong to authenticated client.
- `redirect_uri` must match exactly.
- PKCE verifier must match stored challenge for `S256` or `plain`.

Success response:

```json
{
  "access_token": "...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "...",
  "scope": "openid profile email",
  "id_token": "..."
}
```

`id_token` is included only when final scope contains `openid`.

### Refresh Token Grant

Required form fields:

| Field | Required | Rule |
| --- | --- | --- |
| `grant_type` | yes | `refresh_token` |
| `refresh_token` | yes | Existing active refresh token |

Validation:

- Refresh token must exist.
- Refresh token must not be revoked.
- Refresh token must not be expired.
- Refresh token must belong to authenticated client.
- User must still exist.

Behavior:

- Refresh token is single-use.
- Used refresh token is revoked transactionally.
- New refresh token is issued with the token response.
- ID token is not issued on refresh unless future behavior changes to preserve nonce/profile needs.

## Token Claims

### Access Token

Signing:

- Algorithm: `RS256`.
- Header `kid`: current key ID.
- Expiry: 1 hour.

Claims:

| Claim | Value |
| --- | --- |
| `iss` | configured issuer |
| `sub` | user ID |
| `aud` | array containing client ID |
| `iat` | issued-at Unix timestamp |
| `exp` | expiry Unix timestamp |
| `scope` | final granted scope |
| `role` | user role |

### ID Token

Signing:

- Algorithm: `RS256`.
- Header `kid`: current key ID.
- Expiry: 1 hour.

Base claims:

| Claim | Value |
| --- | --- |
| `iss` | configured issuer |
| `sub` | user ID |
| `aud` | array containing client ID |
| `iat` | issued-at Unix timestamp |
| `exp` | expiry Unix timestamp |

Conditional claims:

| Scope/field | Claim |
| --- | --- |
| `nonce` provided | `nonce` |
| `email` scope | `email` |
| `profile` scope | `name`, `role` |

## Userinfo Endpoint

Endpoint:

```http
GET /userinfo
POST /userinfo
```

Authorization header:

```text
Authorization: Bearer {access_token}
```

Validation:

- Bearer token must be present.
- Token must use RSA signing method.
- Token must validate with current public key.
- Token issuer must match configured `ISSUER`.
- Token must contain `sub`.
- User must still exist.

Success response:

```json
{
  "sub": "user-id",
  "email": "user@example.com",
  "name": "User Name",
  "role": "manager"
}
```

Claim inclusion:

- `sub` always included.
- `email` included only if access token scope contains `email`.
- `name` and `role` included only if scope contains `profile`.

Failure cases:

| Case | Status | Error |
| --- | --- | --- |
| Missing bearer token | `401` | `invalid_token` |
| Invalid token | `401` | `invalid_token` |
| Missing `sub` | `401` | `invalid_token` |
| User not found | `401` | `invalid_token` |
| DB error | `500` | `server_error` |

Requirements:

- Missing/invalid bearer responses include `WWW-Authenticate`.
- Userinfo must not return password hash.
- Userinfo must not return fields outside granted scopes.

## Registration Endpoint

Endpoint:

```http
POST /register
```

Request:

```json
{
  "email": "user@example.com",
  "password": "password123",
  "name": "User Name",
  "role": "staff"
}
```

Validation:

- JSON body must be valid.
- `email`, `password`, and `name` are required.
- Password must be at least 8 characters.
- Email must be unique.
- Role is normalized by DB helper: invalid roles fall back to `staff`.

Success:

```json
{
  "id": "...",
  "email": "user@example.com",
  "name": "User Name",
  "role": "staff"
}
```

Status:

- `201` on success.
- `400` on invalid request.
- `409` on duplicate email.
- `500` on database/hash error.

Security requirements:

- Store bcrypt hash, never plaintext password.
- Do not return password hash.
- Consider disabling public self-registration outside local/demo mode.

## Admin API

Admin APIs require:

```text
X-Admin-Secret: {ADMIN_SECRET}
```

Missing or wrong admin secret returns:

```json
{
  "error": "unauthorized",
  "error_description": "invalid X-Admin-Secret header"
}
```

### `POST /admin/clients`

Request:

```json
{
  "client_id": "mini-bigc-frontend",
  "client_secret": "secret",
  "name": "Mini BigC Frontend",
  "redirect_uris": ["http://localhost:3000/api/auth/callback"],
  "scopes": "openid profile email"
}
```

Validation:

- `client_id`, `client_secret`, `name`, and at least one redirect URI are required.
- `scopes` defaults to `openid profile email`.
- Client secret is bcrypt-hashed.

Success:

- `201`.
- Response includes ID, client ID, name, redirect URIs, and scopes.
- Response must not include client secret hash.

### `GET /admin/clients`

Success:

- `200`.
- Returns clients ordered by `created_at DESC`.
- Does not include client secret hashes.

### `POST /admin/users`

Request:

```json
{
  "email": "manager@example.com",
  "password": "password123",
  "name": "Manager Name",
  "role": "manager"
}
```

Validation:

- Same as `/register`.
- Duplicate email returns `409`.
- Password is bcrypt-hashed.

### `GET /admin/users`

Success:

- `200`.
- Returns users ordered by `created_at DESC`.
- Response includes ID, email, name, and role.
- Response excludes password hash.

### `PATCH /admin/users/{id}/role`

Request:

```json
{
  "role": "manager"
}
```

Validation:

- Path user ID required.
- JSON body required.
- Role must be `manager` or `staff`.
- Missing user returns `404`.

Success:

```json
{
  "id": "user-id",
  "role": "manager"
}
```

## Database Schema

Database migration runs automatically in `db.Open`.

### `users`

| Column | Type | Constraints |
| --- | --- | --- |
| `id` | TEXT | Primary key |
| `email` | TEXT | Unique, not null |
| `password_hash` | TEXT | Not null |
| `name` | TEXT | Not null |
| `role` | TEXT | Not null, default `staff` |
| `created_at` | DATETIME | Not null, default current timestamp |

Rules:

- Role must be `manager` or `staff` at creation/update boundaries.
- Existing databases receive `role` through additive `ALTER TABLE`.
- Password hash must be bcrypt.

### `clients`

| Column | Type | Constraints |
| --- | --- | --- |
| `id` | TEXT | Primary key |
| `client_id` | TEXT | Unique, not null |
| `client_secret_hash` | TEXT | Not null |
| `name` | TEXT | Not null |
| `redirect_uris` | TEXT | JSON-encoded array, not null |
| `scopes` | TEXT | Not null |
| `created_at` | DATETIME | Not null, default current timestamp |

Rules:

- Redirect URIs are stored as JSON array text.
- Client secret hash must never be returned from list/create APIs.
- Client ID uniqueness protects OAuth client identity.

### `authorization_codes`

| Column | Type | Constraints |
| --- | --- | --- |
| `code` | TEXT | Primary key |
| `client_id` | TEXT | Not null |
| `user_id` | TEXT | Not null |
| `redirect_uri` | TEXT | Not null |
| `scope` | TEXT | Not null |
| `nonce` | TEXT | Not null, default empty |
| `code_challenge` | TEXT | Not null, default empty |
| `code_challenge_method` | TEXT | Not null, default empty |
| `expires_at` | DATETIME | Not null |
| `used` | INTEGER | Not null, default `0` |
| `created_at` | DATETIME | Not null, default current timestamp |

Rules:

- Code lifetime is 10 minutes.
- Code must be marked used in the same transaction that reads it.
- Code reuse returns `invalid_grant`.
- Expired code returns `invalid_grant`.

### `refresh_tokens`

| Column | Type | Constraints |
| --- | --- | --- |
| `token` | TEXT | Primary key |
| `client_id` | TEXT | Not null |
| `user_id` | TEXT | Not null |
| `scope` | TEXT | Not null |
| `expires_at` | DATETIME | Not null |
| `revoked` | INTEGER | Not null, default `0` |
| `created_at` | DATETIME | Not null, default current timestamp |

Rules:

- Refresh token lifetime is 30 days.
- Refresh tokens are single-use and revoked transactionally.
- Revoked or expired tokens return `invalid_grant`.

## Security Requirements

Always:

- Hash user passwords with bcrypt.
- Hash client secrets with bcrypt.
- Sign tokens with RS256.
- Protect admin APIs with `X-Admin-Secret`.
- Exact-match redirect URIs.
- Preserve and return `state`.
- Validate PKCE verifier when challenge is present.
- Exclude password/client secret hashes from responses.
- Set token endpoint no-store cache headers.

Never:

- Commit real `private.pem`, production `sso.db`, or production admin secrets.
- Log plaintext passwords or client secrets.
- Return password hashes, client secret hashes, or private key material.
- Accept wildcard redirect URIs.
- Treat self-registration as production-safe without additional controls.

Production gaps to resolve before external use:

- Add CSRF protection for login form.
- Add login rate limiting.
- Add account lockout or throttling.
- Add HTTPS-only deployment.
- Add secure cookie/session support if browser sessions are introduced.
- Add audit logging for admin changes.
- Add key rotation.
- Add refresh token family reuse detection.
- Add proper email verification and password reset.

## Key Management

Current behavior:

- Load `private.pem` if it exists.
- Parse as PKCS#8 RSA private key.
- If no valid key exists, generate a 2048-bit RSA key.
- Write generated key to `private.pem` with mode `0600`.
- Use static `kid` value `key-1`.

Requirements:

- Generated key must remain stable across server restarts to keep tokens verifiable.
- JWKS must expose matching public key.
- Token header must use matching `kid`.
- Key file parse errors should fail startup.

Future requirements:

- Support multiple active keys.
- Support key rotation with overlapping old/new keys.
- Generate deterministic `kid` from key fingerprint or managed metadata.

## OAuth Flow

### Authorization Code With PKCE

```text
Client
→ GET /authorize?client_id=...&redirect_uri=...&response_type=code&scope=openid profile email&state=...&nonce=...&code_challenge=...&code_challenge_method=S256
→ SSO renders login page
→ User submits email/password
→ SSO validates client, redirect URI, user credentials
→ SSO creates auth code
→ 302 redirect_uri?code=...&state=...
→ Client POST /token with client credentials, code, redirect_uri, code_verifier
→ SSO validates code and PKCE
→ SSO issues access token, ID token, refresh token
→ Client calls /userinfo with bearer access token
```

Acceptance criteria:

- Code cannot be used twice.
- Code cannot be exchanged by a different client.
- Code cannot be exchanged with different redirect URI.
- PKCE S256 uses base64url SHA-256 of verifier.
- `nonce` appears in ID token when provided.

### Refresh Token Flow

```text
Client
→ POST /token grant_type=refresh_token
→ SSO validates client and refresh token
→ SSO revokes old refresh token
→ SSO issues new access token and refresh token
```

Acceptance criteria:

- Old refresh token cannot be reused.
- Refresh token from a different client cannot be used.
- Expired refresh token fails.

## Error Contract

OAuth-style JSON error shape:

```json
{
  "error": "invalid_request",
  "error_description": "human-readable detail"
}
```

Use this shape for:

- `/token`
- `/userinfo`
- `/register`
- `/admin/*`

Plain text HTTP errors are currently used for some `/authorize` validation failures before login rendering.

Status guidance:

| Status | Use |
| --- | --- |
| `200` | Discovery, JWKS, userinfo, admin list/update success |
| `201` | Register, admin create user/client |
| `302` | Successful authorize POST |
| `400` | Invalid request, invalid grant, unsupported response/grant |
| `401` | Invalid client, invalid token, login failure, admin unauthorized |
| `404` | Admin role update missing user |
| `409` | Duplicate user email |
| `500` | Database, signing, hashing, key, or server errors |

## Frontend Integration

The Mini BigC frontend should use this SSO service as an OIDC provider.

Frontend integration requirements:

- Use discovery URL from configured issuer.
- Register frontend client with exact callback redirect URI.
- Request `openid profile email`.
- Preserve `state`.
- Use `nonce` for OIDC login.
- Use PKCE for public-browser flows if client secret is not safe to expose.
- Read `role` from ID token profile claim or `/userinfo`.
- Map SSO role to frontend `Role`: `manager` or `staff`.
- Treat tokens as sensitive and avoid logging them.

Expected profile fields:

| SSO field | Frontend use |
| --- | --- |
| `sub` | Employee/user ID fallback |
| `email` | Current user email |
| `name` | Current user name |
| `role` | App access and masking |

## Admin Setup Flow

Local setup sequence:

1. Start SSO server.
2. Create frontend OAuth client with `/admin/clients`.
3. Create at least one manager user with `/admin/users`.
4. Create at least one staff user with `/admin/users`.
5. Configure frontend SSO client settings.
6. Open frontend login/SSO entry.
7. Complete auth code flow.

Example client creation:

```bash
curl -X POST http://localhost:8080/admin/clients \
  -H 'Content-Type: application/json' \
  -H 'X-Admin-Secret: change-me-in-production' \
  -d '{
    "client_id": "mini-bigc-frontend",
    "client_secret": "local-secret",
    "name": "Mini BigC Frontend",
    "redirect_uris": ["http://localhost:3000/api/auth/callback"],
    "scopes": "openid profile email"
  }'
```

Example user creation:

```bash
curl -X POST http://localhost:8080/admin/users \
  -H 'Content-Type: application/json' \
  -H 'X-Admin-Secret: change-me-in-production' \
  -d '{
    "email": "manager@example.com",
    "password": "password123",
    "name": "Branch Manager",
    "role": "manager"
  }'
```

## Testing Strategy

### Unit Tests

Suggested Go unit tests:

- `config.Load` defaults and env overrides.
- `containsURI` exact matching.
- `intersectScope` only returns client-allowed scopes.
- `verifyPKCE` for `S256`, `plain`, empty, and unsupported methods.
- `extractClientCredentials` for Basic and form credentials.
- `checkPassword` with bcrypt hashes.
- `jwkFromRSA` produces required public fields.
- Admin auth accepts only exact configured secret.

Command:

```bash
go test ./...
```

### Handler Tests

Use `httptest` for:

- Discovery returns configured issuer endpoints.
- JWKS returns public key and matching `kid`.
- `GET /authorize` rejects missing parameters.
- `GET /authorize` rejects unknown client.
- `GET /authorize` rejects unregistered redirect URI.
- `POST /authorize` rejects invalid credentials with generic error.
- `POST /authorize` redirects with code and state on success.
- `/token` rejects missing client credentials.
- `/token` rejects reused authorization code.
- `/token` validates PKCE.
- `/userinfo` rejects missing bearer token.
- `/userinfo` returns claims according to scope.
- `/register` validates required fields and duplicate email.
- Admin APIs reject missing secret.
- Admin list APIs do not include hashes.

### Database Tests

Use temporary SQLite files or in-memory SQLite where practical.

Required coverage:

- Migration creates all tables.
- Existing DB without `role` column is upgraded.
- Create/get user.
- Invalid role falls back to `staff` on create.
- Update role rejects invalid role.
- Create/get/list client with redirect URI JSON.
- Auth code is single-use.
- Expired auth code fails.
- Refresh token is single-use.
- Expired refresh token fails.

### End-To-End Flow Tests

Full flow should cover:

1. Create client.
2. Create user.
3. Start authorize GET.
4. Submit authorize POST.
5. Extract authorization code from redirect.
6. Exchange code for tokens.
7. Decode JWT header and verify `kid`.
8. Verify token claims.
9. Call userinfo.
10. Refresh token once.
11. Confirm old refresh token cannot be reused.

## Test Matrix

| Area | Case | Expected |
| --- | --- | --- |
| Discovery | GET metadata | `200`, issuer and endpoints correct |
| JWKS | GET keys | `200`, public RSA key present |
| Authorize | Missing required params | `400` |
| Authorize | Wrong response type | `400` |
| Authorize | Bad redirect URI | `400` |
| Authorize | Wrong password | `401`, generic login error |
| Authorize | Success | `302`, code and state present |
| Token | Missing client | `401 invalid_client` |
| Token | Wrong secret | `401 invalid_client` |
| Token | Missing code | `400 invalid_request` |
| Token | Code reuse | `400 invalid_grant` |
| Token | PKCE mismatch | `400 invalid_grant` |
| Token | Success | `200`, access/refresh/id tokens |
| Userinfo | Missing bearer | `401 invalid_token` |
| Userinfo | Bad token | `401 invalid_token` |
| Userinfo | Email scope absent | No email in response |
| Register | Short password | `400 invalid_request` |
| Register | Duplicate email | `409 conflict` |
| Admin | Missing secret | `401 unauthorized` |
| Admin | Create client | `201`, no secret hash |
| Admin | Create user | `201`, no password hash |
| Admin | Update missing user | `404 not_found` |

## Observability

Current logging:

- Startup logs server address, issuer, discovery URL, and static asset serving.
- Handler errors log database/hash/signing failures.

Requirements:

- Do not log passwords.
- Do not log client secrets.
- Do not log refresh tokens or access tokens.
- Do not log private key contents.
- DB/signing errors may be logged with enough context for debugging.

Recommended future logging fields:

- route
- method
- status
- client_id
- user_id where safe
- error code
- latency

## Performance And Limits

Current local expectations:

- Discovery and JWKS should return in under 100ms locally.
- Token signing should return in under 500ms locally.
- SQLite uses `SetMaxOpenConns(1)` to avoid concurrency issues with local file DB.

Future production considerations:

- Move persistence to a managed DB if concurrency grows.
- Add request body size limits.
- Add rate limits to login, register, token, and admin endpoints.
- Add cleanup job for expired authorization codes and refresh tokens.

## Boundaries

Always:

- Keep OIDC discovery aligned with implemented endpoints.
- Exact-match redirect URIs.
- Hash secrets before storage.
- Use transactions for one-time auth code and refresh token use.
- Keep response hashes/secrets out of API responses.
- Preserve `state` on successful authorization.
- Include `kid` in signed token headers.
- Update this spec when SSO behavior changes.

Ask first:

- Changing issuer or endpoint paths.
- Changing token lifetime.
- Changing refresh token rotation behavior.
- Changing role names.
- Adding public self-registration to production.
- Changing signing algorithm.
- Rotating or deleting existing key material.
- Replacing SQLite with another database.

Never:

- Commit production `private.pem`.
- Commit production `sso.db`.
- Log plaintext credentials.
- Return client secret hash or password hash.
- Accept unregistered redirect URIs.
- Reuse authorization codes.
- Reuse refresh tokens after rotation.
- Treat default admin secret as production-safe.

## Success Criteria

- `go build ./...` passes.
- `go test ./...` passes when tests are present.
- Discovery endpoint returns correct configured issuer URLs.
- JWKS endpoint exposes the public key matching signed JWTs.
- Registered client can complete authorization code flow.
- Token endpoint returns RS256 access token and ID token for `openid` scope.
- Userinfo returns only claims allowed by token scope.
- Refresh token rotation works and prevents reuse.
- Admin APIs require `X-Admin-Secret`.
- Password and client secret hashes are never returned in responses.

## Implementation Checklist

- [ ] Confirm `ISSUER` is correct for the environment.
- [ ] Confirm `ADMIN_SECRET` is not default outside local development.
- [ ] Confirm `private.pem` is present or can be generated with `0600` permissions.
- [ ] Create frontend OAuth client.
- [ ] Create manager and staff test users.
- [ ] Verify discovery and JWKS.
- [ ] Verify authorize code flow.
- [ ] Verify token exchange.
- [ ] Verify userinfo claims and role.
- [ ] Verify refresh token rotation.
- [ ] Add/update tests for changed handler, DB, key, or config behavior.
- [ ] Run `go build ./...`, `go test ./...`, and `go vet ./...`.

## Open Questions

- Should `/register` remain public or move behind admin-only setup?
- Should login sessions/cookies be introduced for better UX across repeated authorizations?
- Should `role` be included in access token, ID token, userinfo, or all three long term?
- Should token audience validation be stricter in `/userinfo`?
- Should refresh tokens be stored hashed instead of plaintext?
- Should authorization codes be stored hashed?
- Should key rotation support multiple JWKS keys?
- Should admin APIs move to bearer-token auth instead of shared secret?
- Should SSO expose a logout/revocation endpoint?
- Should self-service password reset be added?
