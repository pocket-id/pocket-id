# Agent-Compatible Account Completion Code Review Guide

## Purpose

This document maps the implemented agent-compatible account completion feature into discrete review units. Each unit has a sequential `FCAXX` identifier and one matching source comment at its primary implementation anchor.

Use the identifiers to move between this guide and the code:

```sh
rg -n 'FCA[0-9]{2}' backend frontend tests
```

The feature adds a second authentication process for an ordinary Pocket ID user. It does not add a second identity class or an agent-specific authorization policy. `isAgent = false` selects the existing passkey path; `isAgent = true` selects an asymmetric runtime-key path. Both paths produce ordinary Pocket ID session and OIDC state and retain existing groups, claims, disabled state, and administrator behavior.

## Suggested Review Order

| Identifier | Review unit | Primary anchor | Main review concern |
| --- | --- | --- | --- |
| FCA01 | Binary authentication-path selector | `backend/internal/model/user.go` | Authority and identity parity |
| FCA02 | Runtime credential and challenge models | `backend/internal/model/runtime_credential.go` | Secret boundaries and lifecycle data |
| FCA03 | Cross-database schema and enforcement | PostgreSQL runtime migration | Integrity, compatibility, and rollback |
| FCA04 | Module wiring and route authorization | `backend/internal/runtimecredential/module.go` | Public versus authenticated endpoints |
| FCA05 | Shared one-time bootstrap consumption | `backend/internal/onetimeaccess/service.go` | Atomicity and compensation |
| FCA06 | Runtime credential registration | `backend/internal/runtimecredential/service.go` | Bootstrap binding and key validation |
| FCA07 | Repeat runtime authentication | `backend/internal/runtimecredential/service.go` | Proof verification and normal session issuance |
| FCA08 | Fresh reauthentication | `backend/internal/runtimecredential/service.go` | Authenticated-user binding |
| FCA09 | Credential visibility, rename, audit, and revocation | `backend/internal/runtimecredential/service.go` | Ownership and revocation semantics |
| FCA10 | Expired challenge cleanup | `backend/internal/runtimecredential/cleanup.go` | Bounded temporary state |
| FCA11 | WebAuthn path exclusion | `backend/internal/webauthn/service.go` | No alternate credential path |
| FCA12 | Administrator selection and management UI | `frontend/src/routes/settings/admin/users/user-form.svelte` | Accurate controls and backend enforcement |
| FCA13 | User credential-management UI | `frontend/src/routes/settings/account/+page.svelte` | Path-specific presentation and safe metadata |
| FCA14 | Automated acceptance and regression coverage | `tests/specs/runtime-credential.spec.ts` | End-to-end security properties |

## FCA01 — Binary Authentication-Path Selector

### Change

`model.User` and the user API now expose `IsAgent`/`isAgent` as a boolean. The value is false by default and selects the existing passkey path unless an administrator explicitly enables the runtime-key path.

User creation copies the administrator-provided value. LDAP-created or LDAP-synchronized users are forced to the passkey path. User updates treat the selector as an administrator-only field and do not permit self-service changes. Before changing an existing user, the service counts passkeys and active runtime credentials in the same database transaction and returns `authentication_path_change_blocked` if either exists.

The selector has no authorization effect. `IsAdmin`, groups, custom claims, OIDC claims, disabled state, and all existing user behavior remain independent.

### Files to review

- `backend/internal/model/user.go`
- `backend/internal/dto/user_dto.go`
- `backend/internal/service/user_service.go`
- `backend/internal/service/user_service_test.go`
- `backend/internal/apperror/error.go`
- `backend/internal/apperror/constructors.go`
- `frontend/src/lib/types/user.type.ts`

### Review focus

- Only an administrator update path can change `isAgent`.
- LDAP synchronization cannot unexpectedly create a runtime-path user.
- The zero-credential transition test includes every passkey and every non-revoked runtime credential.
- Revoked runtime credentials remain historical records but no longer block a return to the passkey path.
- No authorization branch depends on `isAgent`.

## FCA02 — Runtime Credential and Challenge Models

### Change

Two persistent models support the proof protocol:

- `RuntimeCredential` stores the credential name, algorithm, public key, timestamps, revocation state, and owning user.
- `RuntimeCredentialChallenge` stores short-lived registration, login, or reauthentication proof state.

The implementation accepts Ed25519 only. Pocket ID stores the 32-byte public key; it never accepts, derives, exports, or stores the runtime's private key. Registration challenges temporarily carry the proposed credential name, algorithm, and public key so the durable credential is not created until proof succeeds.

The external DTO returns lifecycle metadata only: ID, name, algorithm, creation time, last-use time, optional expiration, and revocation time. Public-key bytes, challenge bytes, and signatures are excluded from list responses.

### Files to review

- `backend/internal/model/runtime_credential.go`
- `backend/internal/model/user.go`
- `backend/internal/runtimecredential/dto.go`
- `frontend/src/lib/types/runtime-credential.type.ts`

### Review focus

- No private-key field exists at any persistence or API layer.
- Challenge data is sufficient to finish only the operation named by the row.
- Foreign keys attach both records to the normal Pocket ID user.
- Returned metadata is useful for administration without exposing proof material.

## FCA03 — Cross-Database Schema and Enforcement

### Change

Matching PostgreSQL and SQLite migrations add runtime credentials, proof challenges, indexes, foreign keys, algorithm and key-length checks, and database triggers.

The first migration introduced the runtime tables with the initial string-based path marker. The follow-up compatibility migration converts that marker to `users.is_agent`, preserving enabled users while replacing the interim field. This two-step history allows databases created during the first implementation to upgrade safely; a future upstream squash may consolidate the migrations if compatibility with that interim build is unnecessary.

Database constraints enforce:

- Ed25519 as the stored algorithm.
- A 32-byte stored public key.
- One non-revoked runtime credential per user through a partial unique index.
- Allowed challenge operations: `register`, `login`, or `reauthenticate`.
- No passkey insertion for a runtime-path user.
- No runtime credential insertion for a passkey-path user.
- No authentication-path transition while a passkey or active runtime credential exists.

Both migration timelines include down migrations. The SQLite multi-statement migrations explicitly control transactions and foreign-key handling as required by the repository migration runner.

### Files to review

- `backend/resources/migrations/postgres/20260814050000_runtime_credentials.up.sql`
- `backend/resources/migrations/postgres/20260814050000_runtime_credentials.down.sql`
- `backend/resources/migrations/postgres/20260814060000_binary_agent_selector.up.sql`
- `backend/resources/migrations/postgres/20260814060000_binary_agent_selector.down.sql`
- `backend/resources/migrations/sqlite/20260814050000_runtime_credentials.up.sql`
- `backend/resources/migrations/sqlite/20260814050000_runtime_credentials.down.sql`
- `backend/resources/migrations/sqlite/20260814060000_binary_agent_selector.up.sql`
- `backend/resources/migrations/sqlite/20260814060000_binary_agent_selector.down.sql`
- `backend/internal/runtimecredential/migration_test.go`

### Review focus

- PostgreSQL and SQLite enforce equivalent invariants.
- Up and down migrations preserve the selected path across the interim string-to-boolean conversion.
- Partial uniqueness allows historical revoked credentials while limiting active credentials to one.
- Cascading deletes remove runtime credentials and challenges with their user.
- Service checks provide useful application errors while database triggers remain the final race-safe boundary.

## FCA04 — Module Wiring and Route Authorization

### Change

A dedicated `runtimecredential` module owns DTO binding, handlers, proof service logic, and route registration. Bootstrap wiring injects existing Pocket ID services for database access, JWT/session issuance, audit logging, one-time-token consumption, reauthentication-token issuance, and app configuration.

Routes are divided by authority:

- Registration start/finish and login start/finish are public but rate-limited.
- Reauthentication start/finish require browser user authentication, disallow API-key substitution, and use the existing reauthentication rate limiter.
- List, rename, and revoke for the current user require normal user authentication.
- Per-user list and revoke routes require administrator authentication.

The public proof endpoints still require possession of either the administrator-issued bootstrap token or the runtime private key. Public error handling intentionally avoids an account-specific credential oracle.

### Files to review

- `backend/internal/runtimecredential/module.go`
- `backend/internal/runtimecredential/handler.go`
- `backend/internal/bootstrap/services_bootstrap.go`
- `backend/internal/bootstrap/router_bootstrap.go`
- `backend/internal/apperror/error.go`
- `backend/internal/apperror/constructors.go`

### Review focus

- Middleware order and authentication requirements match each route's authority.
- Public proof endpoints inherit an appropriate existing rate limiter.
- API keys cannot stand in for browser authentication during fresh reauthentication.
- Handlers set the existing access-token and reauthentication cookies rather than inventing parallel session state.

## FCA05 — Shared One-Time Bootstrap Consumption

### Change

The one-time-access module now exposes atomic token consumption and best-effort restoration to other completion flows. Ordinary one-time login and runtime credential registration therefore compete for the same one-time authority; either path consumes the administrator-issued token.

Consumption invokes the Francis token actor before opening a database transaction. This preserves atomic single use and avoids deadlocking SQLite by invoking an actor while a database transaction is open. If runtime registration cannot create its durable challenge after consumption, the service restores the token state as compensation.

Once a registration challenge is stored successfully, the bootstrap token remains consumed. An abandoned registration expires with the challenge and requires a new administrator-issued link.

### Files to review

- `backend/internal/onetimeaccess/module.go`
- `backend/internal/onetimeaccess/service.go`
- `backend/internal/runtimecredential/module.go`
- `backend/internal/runtimecredential/service.go`

### Review focus

- Token consumption remains atomic across ordinary exchange and runtime registration.
- Restoration occurs only when registration fails before the challenge is durably committed.
- Device-token validation retains its existing behavior.
- No actor invocation occurs while a SQLite transaction is open.

## FCA06 — Runtime Credential Registration

### Change

Registration is a two-step proof-of-possession flow.

`BeginRegistration` normalizes and consumes the one-time token, validates an unpadded-base64url Ed25519 public key, locks and validates the user, confirms `isAgent = true`, rejects disabled users, rejects existing passkeys, and rejects a second active runtime credential. It then stores a 60-second `register` challenge with the proposed name and public key.

`FinishRegistration` atomically deletes and returns the unexpired registration challenge inside a transaction, rechecks the user and credential-family state, verifies the Ed25519 signature, creates the durable credential, emits registration and normal sign-in audit events, and generates the ordinary Pocket ID access token and cookie.

Challenge bytes are built as an operation-specific protocol prefix, a newline, and 32 cryptographically random bytes. Wire values use unpadded base64url. The client signs the decoded challenge bytes exactly.

### Files to review

- `backend/internal/runtimecredential/dto.go`
- `backend/internal/runtimecredential/service.go`
- `backend/internal/runtimecredential/handler.go`
- `backend/internal/runtimecredential/service_test.go`

### Review focus

- The private key is generated and retained entirely by the client.
- State is revalidated at finish rather than trusted from the start request.
- A successful challenge cannot be replayed because deletion and credential creation commit together.
- Failed proof rolls back the transactional deletion, permitting a correctly signed retry only until challenge expiry and subject to rate limiting.
- The one-active-credential rule is enforced by service logic and the partial unique index.

## FCA07 — Repeat Runtime Authentication

### Change

Repeat login begins with the normal Pocket ID username and stored credential ID. The pair locates a non-revoked credential without introducing a second user identifier. Invalid username, credential ID, disabled user, wrong path, expiration, expired challenge, or invalid signature returns the same generic runtime-credential error.

The finish step atomically consumes the `login` challenge, locks and reloads the credential and user, rechecks revocation, expiration, disabled state, ownership, and authentication path, verifies the signature, updates `last_used_at`, and records a runtime-authentication audit event.

Successful proof generates the existing Pocket ID access token with authentication method `pop`. The handler sets the same access-token cookie used by the existing browser and OIDC flows. Downstream subject and configured claims continue through the existing user and OIDC pipeline.

### Files to review

- `backend/internal/runtimecredential/dto.go`
- `backend/internal/runtimecredential/service.go`
- `backend/internal/runtimecredential/handler.go`
- `backend/internal/model/audit_log.go`
- `backend/internal/oidc/preview_test.go`

### Review focus

- Credential lookup requires both username and credential ID and does not reveal which input failed.
- Revocation and disabled state are checked at both start and finish.
- Challenge operation binding prevents using a registration or reauthentication challenge for login.
- `amr = ["pop"]` reports generic proof of possession without claiming phishing resistance, hardware custody, or human presence.
- Normal OIDC claims are unchanged.

## FCA08 — Fresh Reauthentication

### Change

Runtime-path users can satisfy Pocket ID's existing fresh-authentication requirement through a runtime proof flow. Start requires an authenticated browser session and queries the requested credential by both credential ID and current user ID. Finish repeats the credential proof checks and also verifies that the challenge-resolved user equals the authenticated user before calling the existing reauthentication-token issuer.

The resulting cookie and freshness semantics are the same ones used after a fresh passkey ceremony. The authentication evidence differs, but authorization and the protected operation remain unchanged.

### Files to review

- `backend/internal/runtimecredential/module.go`
- `backend/internal/runtimecredential/handler.go`
- `backend/internal/runtimecredential/service.go`
- `backend/internal/webauthn/module.go`
- `backend/internal/webauthn/service.go`

### Review focus

- A credential owned by another user cannot satisfy reauthentication.
- Reauthentication cannot be performed through API-key authentication.
- The feature reuses the existing reauthentication token rather than adding a parallel bypass.
- Runtime proof does not claim WebAuthn-specific assurance.

## FCA09 — Credential Visibility, Rename, Audit, and Revocation

### Change

Authenticated users can list, rename, and revoke their own runtime credentials. Administrators can list and revoke credentials for a specified user. Every query scopes the credential by both credential ID and user ID.

Registration, successful proof authentication, and revocation have distinct audit events. Revocation records credential metadata and includes administrator actor metadata when the actor differs from the credential owner. Rename changes display metadata only and does not currently emit a dedicated audit event.

Revocation sets `revoked_at` and immediately prevents future challenge creation or proof completion. It intentionally does not invalidate already-issued Pocket ID cookies, access tokens, refresh tokens, OAuth grants, or relying-party sessions. Those continue under existing logout and per-client lifetime controls.

### Files to review

- `backend/internal/runtimecredential/service.go`
- `backend/internal/runtimecredential/handler.go`
- `backend/internal/runtimecredential/dto.go`
- `backend/internal/model/audit_log.go`
- `frontend/src/lib/services/runtime-credential-service.ts`

### Review focus

- Self-service operations cannot address another user's credential.
- Administrator routes use admin middleware and record the acting administrator on revocation.
- Repeated revocation is idempotent.
- Revoked records remain visible for operational history.
- Existing-session survival after revocation matches Pocket ID's existing token model and is clearly communicated in the UI and documentation.
- Reviewers should decide whether rename requires a future audit event; the current implementation does not add one.

## FCA10 — Expired Challenge Cleanup

### Change

Expired proof challenges are deleted by a new database cleanup function registered with the existing scheduled cleanup jobs. The job runs immediately on scheduler start and then on the same jittered daily cadence used for similar temporary authentication state, with the existing exponential-backoff behavior.

Authentication correctness does not depend on cleanup timing because every challenge lookup also requires `expires_at > now`. Cleanup bounds retained expired data rather than enforcing expiration.

### Files to review

- `backend/internal/runtimecredential/cleanup.go`
- `backend/internal/job/db_cleanup_job.go`

### Review focus

- Expired challenges are unusable before cleanup runs.
- Cleanup deletes only expired runtime challenge rows.
- Transient cleanup failures use existing scheduler retry behavior.

## FCA11 — WebAuthn Path Exclusion

### Change

The WebAuthn service rejects runtime-path users during passkey registration start, registration finish, discoverable login, and fresh reauthentication. These service checks provide stable application errors and complement the database trigger that prevents passkey insertion.

This is a credential-path safeguard, not an assertion about whether the operator is human. Anyone operating the account is bound to its configured authentication path. A runtime-path administrator cannot silently add a passkey as an alternate login route.

### Files to review

- `backend/internal/webauthn/service.go`
- `backend/internal/webauthn/service_test.go`
- `backend/resources/migrations/postgres/20260814060000_binary_agent_selector.up.sql`
- `backend/resources/migrations/sqlite/20260814060000_binary_agent_selector.up.sql`

### Review focus

- All WebAuthn entry points enforce the selector, not only registration.
- Database enforcement closes races and non-service insertion paths.
- Passkey-path behavior remains unchanged when `isAgent` is false.

## FCA12 — Administrator Selection and Management UI

### Change

The administrator user form adds an **Agent authentication** switch backed by `isAgent`. The standard username remains the only user-facing sign-in identifier. The edit screen disables the switch and explains why when any passkey or active runtime credential exists.

The administrator credential tab selects either passkeys or runtime credentials according to the configured path. Runtime credential rows show name, ID, algorithm, creation time, last use, expiration, and revocation state, with an administrator revoke action. Revocation messaging explicitly describes future-authentication-only behavior.

The UI is advisory. The user service and database triggers independently enforce every path transition and credential-family rule.

### Files to review

- `frontend/src/routes/settings/admin/users/user-form.svelte`
- `frontend/src/routes/settings/admin/users/[id]/+page.ts`
- `frontend/src/routes/settings/admin/users/[id]/+page.svelte`
- `frontend/src/routes/settings/admin/users/[id]/admin-runtime-credential-list.svelte`
- `frontend/src/lib/components/runtime-credential-list.svelte`
- `frontend/src/lib/services/runtime-credential-service.ts`
- `frontend/messages/en.json`

### Review focus

- The selector defaults to off and is represented as a boolean, not free text.
- Revoked credentials do not block the selector in the UI.
- The credential tab never offers passkey creation for a runtime-path user.
- Text does not imply immediate invalidation of existing sessions or OAuth state.
- Generated Paraglide files are not hand-edited.

## FCA13 — User Credential-Management UI

### Change

The account page loads and displays only the credential family selected for the current user. Passkey-path users retain the existing passkey warnings and add/rename/delete interactions. Runtime-path users see runtime credential metadata, rename and revoke controls, and a warning to request a new one-time link when no active credential remains.

A shared list component renders safe lifecycle metadata for both self-service and administrator views. Small wrapper components supply the authority-specific revoke operation, and the self-service view additionally supplies rename.

### Files to review

- `frontend/src/routes/settings/account/+page.ts`
- `frontend/src/routes/settings/account/+page.svelte`
- `frontend/src/routes/settings/account/runtime-credential-list.svelte`
- `frontend/src/routes/settings/account/rename-runtime-credential-modal.svelte`
- `frontend/src/lib/components/runtime-credential-list.svelte`
- `frontend/src/lib/services/runtime-credential-service.ts`
- `frontend/src/lib/types/runtime-credential.type.ts`

### Review focus

- Runtime-path users are not offered passkey creation controls.
- No public-key or proof material appears in frontend types or responses.
- Revoking the last active credential leaves a clear recovery path through administrator-issued bootstrap.
- Account editing cannot change `isAgent`.

## FCA14 — Automated Acceptance and Regression Coverage

### Change

Backend tests cover registration, key validation, one-active-credential enforcement, login, generic failure behavior, reauthentication, rename, listing, revocation, audit events, database triggers, path-transition blocking, WebAuthn exclusion, OIDC claim parity, and `amr = ["pop"]`.

Migration tests cover latest-schema down/up behavior and conversion of a user created with the interim string marker to `is_agent = true`.

The Playwright scenario uses a cookie-empty client, generates an Ed25519 keypair in process memory, registers through an administrator-issued one-time token, proves registration-challenge replay denial, exercises an existing administrator API with the runtime-authenticated session, performs repeat login, revokes the credential as an administrator, confirms the existing session survives, and confirms a new login is denied.

The scenario passes on rebuilt SQLite and PostgreSQL stacks. PostgreSQL was additionally exercised through the binary-selector down/up migration round trip with value preservation.

### Files to review

- `backend/internal/runtimecredential/service_test.go`
- `backend/internal/runtimecredential/migration_test.go`
- `backend/internal/service/user_service_test.go`
- `backend/internal/webauthn/service_test.go`
- `backend/internal/oidc/preview_test.go`
- `tests/specs/runtime-credential.spec.ts`
- `tests/resources/export/database.json`

### Review focus

- The synthetic client's private key never leaves process memory.
- Tests distinguish future-authentication denial from existing-session invalidation.
- Both service and database enforcement are exercised.
- OIDC assertions prove identity and claim parity rather than introducing agent-specific output.

## Validation Summary

The implemented change has been validated with:

```text
go test -tags=exclude_frontend,unit ./...                                  PASS
pnpm check                                                                 PASS, 0 errors and 0 warnings
authored frontend Prettier and ESLint                                      PASS
Docker production build                                                    PASS
Playwright runtime-credential.spec.ts on rebuilt SQLite                    PASS
Playwright runtime-credential.spec.ts on rebuilt PostgreSQL                PASS
PostgreSQL binary-selector down/up round trip and scenario rerun           PASS
```

The review identifiers are navigation aids only. They do not affect runtime behavior, API contracts, persistence, or test selection.
