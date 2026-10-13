# Development Environment

## Requirements

| Tool | Version | Notes |
|---|---|---|
| Go | **1.26+** | everything else (task, sqlc, golangci-lint, air) is pinned and runs through `go tool` |
| Docker | any recent | only for the dev database (`task db:up`) and image builds |
| PostgreSQL | **18** | provided by `compose.dev.yaml`; `uuidv7()` needs 18 |

No global installs. `go tool task <name>` runs a task (alias it: `alias t='go tool task'`).
Dev tools live in `tools/go.mod` so they never mix with the app's dependencies.

> **Docker context.** `docker` commands go to the *current context*. If it points at a remote host
> (`docker context ls`), `task db:up` starts Postgres there and bind mounts of `db/init` will fail.
> Use a local engine (`docker context use desktop-linux` / `default`) for development, or point
> `NETOPOLOGY_DATABASE_URL` / `NETOPOLOGY_TEST_ADMIN_URL` at any Postgres 18 you prefer.

## First run (backend alone)

```sh
go tool task setup     # .env from .env.example, download modules
go tool task db:up     # Postgres 18 on 127.0.0.1:5432, roles created on first start
go tool task dev       # api on :8080 with live reload; migrations run on start
# in another terminal, if you need background jobs:
go tool task worker
```

- Docs: http://localhost:8080/api/v1/docs (Scalar, development only). Spec: `/api/v1/openapi.json`.
- Probes: `/healthz` (process up), `/readyz` (database reachable).

## Frontend + backend together

In the parent repo (frontend at the root), with the backend running on :8080:

```sh
# parent repo, .env.development.local
NEXT_PUBLIC_API_MOCKING=disabled
NETOPOLOGY_API_URL=http://localhost:8080
```
`pnpm dev` then proxies `/api/*` to the backend (next.config.ts rewrites), so the browser sees one
origin (http://localhost:3000), which must equal `NETOPOLOGY_PUBLIC_URL`.

The full production-like stack (gateway + web + api + worker + db) is `docker compose up` in the
parent repo (`compose.yaml`).

## Everyday tasks

| Task | Does |
|---|---|
| `task gen` | sqlc + `api/openapi.gen.yaml`. Run after changing SQL or operations. |
| `task test` | unit + contract + integration (when `NETOPOLOGY_TEST_ADMIN_URL` is set) |
| `task lint` / `task fmt` | golangci-lint v2 (gofumpt, goimports, gosec, depguard layering rule, ...) |
| `task check` | gen + "generated code committed" + lint + test. **Run before every commit.** |
| `task migrate` / `task migrate:status` | apply / list migrations |
| `task db:psql` | psql as the app role (RLS active: `SELECT set_config('app.workspace_id', '<id>', false);` first) |
| `task db:scenarios` | schema checks |
| `task db:reset` | wipe the dev database |
| `task docker:build` | build the image |

## Environment variables

| Variable | Default | Meaning |
|---|---|---|
| `NETOPOLOGY_ENV` | `production` | `development` enables text logs and `/api/v1/docs` |
| `NETOPOLOGY_LOG_LEVEL` | `info` | `debug` `info` `warn` `error` |
| `NETOPOLOGY_ADDR` | `:8080` | api listen address |
| `NETOPOLOGY_PUBLIC_URL` | **required** | URL users open (OAuth redirects, Origin check, share URLs) |
| `NETOPOLOGY_COOKIE_SECURE` | `true` | `Secure` cookie flag (fine on http://localhost) |
| `NETOPOLOGY_TRUSTED_PROXIES` | private ranges | CIDRs whose `X-Forwarded-For` is trusted (the gateway) |
| `NETOPOLOGY_DATABASE_URL` | **required** | app role (`netopology_app`), RLS applies; superuser is refused |
| `NETOPOLOGY_MIGRATE_DATABASE_URL` | — | schema-owner role; required for `migrate` and when migrating on start |
| `NETOPOLOGY_DB_MAX_CONNS` | `20` | pool size per process |
| `NETOPOLOGY_MIGRATE_ON_START` | `true` | api migrates before serving (advisory-locked) |
| `NETOPOLOGY_SECRET_KEY` | **required** | 32 random bytes, base64 (`openssl rand -base64 32`). Signs cookies; encrypts credentials from C3. |
| `NETOPOLOGY_SESSION_TTL` | `168h` | sliding session lifetime (min 1h) |
| `NETOPOLOGY_GOOGLE_CLIENT_ID` / `_SECRET` | — | enable Google sign-in (both or neither) |
| `NETOPOLOGY_DISCORD_CLIENT_ID` / `_SECRET` | — | enable Discord sign-in (both or neither) |
| `NETOPOLOGY_WORKER_CONCURRENCY` | `10` | parallel jobs in the worker |
| `NETOPOLOGY_TEST_ADMIN_URL` | — | tests only: Postgres 18 **superuser** URL |

Database container only (`compose.dev.yaml`, parent `compose.yaml`): `NETOPOLOGY_DB_OWNER_PASSWORD`,
`NETOPOLOGY_DB_APP_PASSWORD`, `NETOPOLOGY_DB_SUPERUSER_PASSWORD`, `NETOPOLOGY_DB_PORT`.

## Ports

| Port | What |
|---|---|
| 3000 | Next.js dev server (parent repo) |
| 5432 | dev Postgres (127.0.0.1 only) |
| 8080 | api |
| 80/443 | gateway in the full stack |
