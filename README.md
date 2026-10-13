# Netopology Backend

The Go backend of [Netopology](https://github.com/Whitedragon115/Netopology): the API, the background
worker and the database schema. It is included in the main repository as the `backend/` submodule.

**Stack:** Go 1.26 · chi + Huma (OpenAPI 3.1) · PostgreSQL 18 with row-level security · pgx + sqlc ·
goose migrations · River (Postgres job queue) · server-sent events.

## Quick start

```sh
go tool task setup     # .env from .env.example
go tool task db:up     # Postgres 18 in Docker
go tool task dev       # api on http://localhost:8080 (docs at /api/v1/docs)
```

Everything is explained in [docs/ENVIRONMENT.md](docs/ENVIRONMENT.md).

## Documents

| | |
|---|---|
| [AGENTS.md](AGENTS.md) | Rules and workflow for anyone (human or agent) writing code here |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | How the backend is built, and why |
| [docs/WORK_PACKAGES.md](docs/WORK_PACKAGES.md) | The implementation plan and its status |
| [docs/TESTING.md](docs/TESTING.md) | Test kinds and how to write them |
| [docs/ENVIRONMENT.md](docs/ENVIRONMENT.md) | Tools, tasks, environment variables |
| [api/contract.yaml](api/contract.yaml) | The API contract the frontend is built against |

## License

Apache-2.0
