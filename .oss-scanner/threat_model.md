# Pocket ID threat model

This file guides Anthropic's OSS Scanner. Confirm paths and setting names against the scanned checkout; the code moves faster than this document.

## 1. What Pocket ID is

Pocket ID is a self-hosted OpenID Connect provider where users sign in with passkeys only, so there are no passwords to steal.
Operators put it in front of their other services (Nextcloud, Immich, Proxmox, reverse-proxy auth and so on), which makes a compromise of Pocket ID a compromise of everything that trusts it.

- `backend/` is a Go server (gin, GORM, SQLite or Postgres) that serves the API, the OIDC endpoints and the embedded SPA on one port (default 1411).
- OAuth and OIDC come from a fork of ory/fosite (`github.com/pocket-id/fosite`), composed in `backend/internal/oidc/provider.go`.
- `frontend/` is a SvelteKit SPA, built into `backend/frontend/dist` and embedded with `go:embed`.
- Background work (cron jobs, rate limiters, one-time codes, signup tokens, LDAP and SCIM sync) runs as compiled-in Go actors on the Francis actor runtime, stored in the same database.

## 2. Principals and trust boundaries

From least to most trusted:

1. **Anonymous network attacker**: reaches every `.Public()` and `.Optional()` route, the OIDC endpoints and the SPA.
2. **OIDC client**: a relying party with a client ID and, if confidential, a client secret or a federated client assertion; it may be malicious or compromised.
3. **Signed-in user**: holds a session cookie and the `account:*` scopes; may be restricted to `unverifiedScopes` while email verification is pending.
4. **API key**: sent as `X-API-Key`, inherits its owner's scopes except the session-only ones (`account:session`, `account:authorize`, `account:passkeys:enroll`, `account:api-keys:create`).
5. **Admin**: a user with `IsAdmin`, who manages users, groups, OIDC clients, API resources and the application configuration.
6. **Operator**: controls environment variables, the database, the storage backend and the `pocket-id` CLI (`import`, `export`, `key-rotate`, `encryption-key-rotate`, `one-time-access-token`).

Authorization lives in `backend/internal/authz/` (`scope.go`, `principal.go`, `middleware.go`, `router.go`).
Every route declares a scope or is explicitly public, and the user is reloaded on every request so disabling a user or removing admin takes effect at once.
OAuth access tokens issued to relying parties cannot call the management `/api`.

## 3. Attack surface

### OAuth 2.0 and OIDC (`backend/internal/oidc/`)

- `/authorize`, `/api/oidc/par`, `/api/oidc/token`, `/api/oidc/userinfo`, `/api/oidc/introspect`, `/api/oidc/device/*`, `/api/oidc/end-session`, the interaction endpoints and the `/.well-known/*` documents.
- Grants: authorization code with PKCE, refresh token (with reuse detection), device code and client credentials.
- Redirect URI matching, including wildcards, lives in `backend/internal/utils/callback_url_util.go`.
- Client ID Metadata Documents (`oidc/cimd.go`) and federated client authentication (`oidc/federated_client_auth.go`) fetch remote documents and JWKS.
- Back-channel logout (`backend/internal/backchannellogout/`) sends signed logout tokens to relying parties.
- Group restrictions on clients, custom claims and API resources and permissions (`backend/internal/api/`, `oidc/api_resource.go`) decide what a user may authorize and what ends up in tokens.

### Sign-in and account recovery

- Passkeys (`backend/internal/webauthn/`): login, registration, re-authentication for sensitive clients; the RP ID and origin come from `APP_URL`.
- One-time access codes and email login codes (`backend/internal/onetimeaccess/`), including the unauthenticated email request when an admin enables it.
- Cross-device login (`backend/internal/devicelogin/`).
- Self-signup and signup tokens (`backend/internal/usersignup/`), and the initial admin setup at `/api/signup/setup`.
- Email verification and email change (`backend/internal/emailverification/`).
- Sessions are HS256 JWTs in the `__Host-access_token` cookie (`service/jwt_service.go`, `utils/cookie/`).

### Management API

- Users, groups, OIDC clients, API keys (`backend/internal/apikey/`), application configuration (`backend/internal/appconfig/`) and audit logs (`backend/internal/auditlogs/`).
- `STATIC_API_KEY` authenticates as a synthetic admin when an operator sets it.

### Uploads and rendered content

- Application images, client logos and profile pictures (`controller/app_images_controller.go`, `controller/oidc_controller.go`, `controller/user_controller.go`) accept PNG, JPEG, ICO and SVG and are served from the same origin.
- Storage backends are the filesystem (`os.Root`), S3 or the database (`backend/internal/storage/`).
- The SPA renders user-controlled names, client names and URLs, and handles `redirect` parameters on the login pages.

### Outbound requests (`backend/internal/outbound/`)

- Client logo URLs, CIMD documents, federated JWKS, SCIM provisioning (`backend/internal/scimsync/`), back-channel logout and LDAP profile pictures go through a guarded transport that checks each dialed IP and every redirect.
- The `OUTBOUND_ALLOWED_HOSTS_*` variables let operators allow more destinations per purpose.

### LDAP sync (`backend/internal/ldapsync/`)

- Users and groups, including admin status, are imported from an admin-configured directory, so directory data is semi-trusted input.

### Web hardening

- CSRF protection through `http.CrossOriginProtection` with `APP_URL` as the trusted origin, a nonce-based CSP (`utils/csp.go`), CORS only on the endpoints relying parties call from browsers (`middleware/cors.go`) and per-IP rate limits (`middleware/rate_limit.go`).
- Client IPs come from the socket unless `TRUST_PROXY`, `TRUSTED_PLATFORM` or `PROXY_PROTOCOL` say otherwise.

## 4. Secrets and data worth protecting

- `ENCRYPTION_KEY` is the root secret: it encrypts the signing keys and the session key at rest, derives the fosite HMAC secret for codes and refresh tokens, encrypts SCIM bearer tokens and keys the actor runtime's mTLS.
- The signing key (RS256 by default) signs ID tokens and JWT access tokens; the session key signs Pocket ID's own sessions.
- Client secrets and API keys are stored hashed, refresh tokens and codes as fosite HMAC signatures.
- Passkey credentials, user profiles, group memberships, audit logs and SMTP and LDAP credentials in the application configuration.

## 5. How to exercise it

The image has the checkout at `/src`, Go modules and the pnpm store pre-fetched and `GOPROXY=off`; there is no network.

- `/src/backend/pocket-id` is the release build with the SPA embedded.
- `/src/backend/pocket-id-e2etest` adds the end-to-end routes: `POST /api/test/reset?skip-ldap=true` seeds users (`tim` is an admin, `craig` and `eddy` are not), groups and OIDC clients, and `POST /api/test/accesstoken` signs tokens for them; see `tests/data.ts` for the seed data.
- Start a server with `cd /tmp && ENCRYPTION_KEY=test-encryption-key APP_URL=http://localhost:1411 APP_ENV=test /src/backend/pocket-id-e2etest`; data goes to `./data` with SQLite, and the seed's signing key is encrypted with exactly that `ENCRYPTION_KEY`.
- Passkeys cannot be used without a browser and authenticator, so sign in as a user with `pocket-id one-time-access-token <username>` (same env vars) and exchange the code with `POST /api/one-time-access-token/<code>`, which sets the session cookie.
- Unit tests: `cd /src/backend && go test -tags exclude_frontend,unit ./...`; tests that need Postgres skip without `POCKET_ID_TEST_POSTGRES_URL`, and no Postgres is installed.
- Regression tests belong next to the code they test, as `*_test.go` with the `unit` build tag where test helpers are needed; `backend/internal/utils/testing/` has database and actor helpers.
- The Playwright suite in `tests/` needs Docker and a browser and cannot run here.

## 6. Report requirements

- State the commit, the lowest principal from section 2 that can trigger the issue and any non-default setting it needs.
- Give a reproducer against a running server (an HTTP request sequence) or a failing Go test, and a patch with a regression test where possible.
- Trace the route and its scope declaration in `authz` before claiming a missing check, and say whether the result was demonstrated or only read from source.
- One report per root cause; report dependency issues only with a path reachable through Pocket ID.

## 7. Severity

- **Critical**: an anonymous attacker or a malicious OIDC client signs in as another user, obtains tokens for another user, becomes admin, forges tokens or reads `ENCRYPTION_KEY`, signing keys or the session key; remote code execution.
- **High**: a regular user gains admin or another user's account; a client receives claims, scopes or groups the user did not grant or is not allowed; redirect URI validation lets codes or tokens reach an attacker; stored XSS that runs in an admin's session; SSRF through the guarded transport to an address the policy blocks with default settings; a bypass of a passkey, user verification or re-authentication requirement.
- **Medium**: account or email enumeration, bypassing rate limits on code entry, CSRF on a state-changing request, information disclosure of other users' profile data, refresh-token reuse detection bypass, denial of service by an unauthenticated attacker with modest traffic.
- **Low**: issues that need an admin or a non-default insecure setting, self-XSS, missing hardening without a demonstrated impact, timing differences without a practical attack.

## 8. Out of scope

- Admins and operators are trusted: managing users, clients and configuration, uploading images, configuring LDAP, SCIM or SMTP and running the CLI are not vulnerabilities unless a less privileged principal gains those powers.
- Whoever completes `/api/signup/setup` on a fresh instance becomes the first admin by design; races or bypasses after an admin exists are in scope.
- The `e2etest` build tag and its unauthenticated routes exist only for the test suite and are never in release builds.
- Settings that knowingly weaken security (`TRUST_PROXY=true`, `DISABLE_RATE_LIMITING`, `*_SKIP_CERT_VERIFY`, `OUTBOUND_ALLOWED_HOSTS_*`, `STATIC_API_KEY`, non-production `APP_ENV`) cap a finding at low unless the default configuration is affected.
- Generated code (`frontend/src/lib/paraglide/`, `backend/frontend/dist/`, `backend/resources/email-templates/`), the Playwright suite in `tests/`, CI and developer tooling.
- Volumetric denial of service and missing security headers on static assets.
