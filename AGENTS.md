# AGENTS.md

Pocket ID — a passkey-only OIDC provider. Go backend serves a SvelteKit SPA (embedded in
the binary for production). This file lists what isn't obvious from reading the code.

## Layout

- `backend/` — Go module (gin, GORM, [ory/fosite fork](#backend-go)). Its own toolchain, not part of the repository.
- `frontend/` — SvelteKit 5 SPA. Builds into `backend/frontend/dist` and is embedded via `go:embed`.
- `tests/` — Playwright end-to-end tests (drives a Dockerized full stack).

## Build / test / lint

```sh
# backend/   — the exclude_frontend tag is mandatory locally; CI uses it too
go test -tags=exclude_frontend ./...                          # unit/integration tests
go test -tags=exclude_frontend -run TestName ./internal/...   # a single test
golangci-lint run --build-tags=exclude_frontend               # lint (config: backend/.golangci.yml)

# frontend/  (or root)
pnpm check        # svelte-check — the ONLY frontend type gate (no unit tests exist)
pnpm lint         # prettier --check && eslint  (note: not enforced by CI)
pnpm format       # prettier --write — REQUIRED before opening a PR
```

End-to-end (needs Docker; **stop any local backend on `:1411` first** — see gotchas):

```sh
cd tests/setup && docker compose up -d --build   # rebuild after ANY code change, or you test stale code
cd ../.. && pnpm test                            # = playwright test in tests/
```

## Critical gotchas

- **`exclude_frontend` build tag.** `backend/frontend/` embeds `dist/` via `go:embed`. Without
  a built frontend present, plain `go run`/`go test`/`golangci-lint` fail to compile. Always pass
  `-tags exclude_frontend` for backend dev/test/lint. Production builds omit it to embed the SPA.
- **Never edit generated files:** `frontend/src/lib/paraglide/**` (Paraglide i18n output) and
  `backend/frontend/dist/**`. For i18n, only edit `frontend/messages/en.json`; other locales come
  from Crowdin.
- **Migrations are split by DB.** Raw SQL via golang-migrate in
  `backend/resources/migrations/{sqlite,postgres}/` — separate files _and_ separate version
  timelines. Add a matching up/down pair to **both**. Not GORM AutoMigrate. SQLite migrations
  are not auto-wrapped in a transaction (`NoTxWrap`); wrap multi-statement ones manually
  (`PRAGMA foreign_keys=OFF; BEGIN; … COMMIT; PRAGMA foreign_keys=ON;`).

## Backend (Go)

- **Config:** global `common.EnvConfig` (caarlos0/env); any secret var supports a `*_FILE` variant.
- **Logging:** stdlib `log/slog` only (bridged to OpenTelemetry). No zerolog/logrus in app code.
- `go.mod` pins a **fork** of fosite (`replace github.com/ory/fosite => github.com/pocket-id/fosite`).

## Frontend (SvelteKit)

- **Svelte 5 runes only:** `$state`, `$derived`, `$props`, `$bindable`. No `export let`. Event
  modifiers are gone — use `preventDefault` from `$lib/utils/event-util` (`onsubmit={preventDefault(fn)}`).
- **Forms:** use the custom `createForm(schema, initial)` from `$lib/utils/form-util.ts` with
  `form-input.svelte`. The vendored shadcn formsnap/superforms wrappers exist but app forms don't
  use them — match the surrounding file. Import zod as `import { z } from 'zod/v4'`.

## Coding Style Guidelines

### Comments

- Exactly one sentence per line
- There is NO maximum line width: never wrap a single sentence across multiple comment lines, no matter how long that sentence is
- A new line in a comment means a new sentence; a wrapped line does not exist
- No trailing period on single-line comments
- Prefer comments that explain intent, invariants, or why a branch exists
- Avoid comments that simply restate the next line of code
- For multi-step logic, use short section comments to separate the steps and explain why each step exists
- Inside a function, put a one-sentence comment above each major action; the comments double as visual separators between sections and should say what the step does and why, not how
- Favor a few well-placed section comments over a wall of code; a reader should be able to skim the comments and understand the method's flow

```go
// Wrong — one sentence wrapped across multiple lines
// This function performs the main validation logic. It checks
// the input against the schema and returns an error if the
// input is invalid.

// Wrong — trailing period on single-line comment
// Validate the input.

// Right — one sentence per line, each line as long as it needs to be
// This function performs the main validation logic
// It checks the input against the schema and returns an error if the input is invalid

// Right
// Validate the input

// Right
// Normalize the request host so callers can pass either Host or X-Forwarded-Host values

// Right
// Browsers do not accept a cookie Domain attribute set to an IP address
// Returning an empty domain tells the caller to set a host-only cookie instead

// Wrong — restates the code
// Trim whitespace and lowercase the host
host = strings.TrimSpace(strings.ToLower(host))
```

Section comments inside a function — one sentence per major action, describing what and why, acting as visual separators:
