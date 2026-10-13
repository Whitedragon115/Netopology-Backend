# Instructions for coding agents

You are implementing the Netopology backend (Go). The architecture, environment and the reference
feature are done; your job is one **work package** at a time from
[docs/WORK_PACKAGES.md](docs/WORK_PACKAGES.md).

## Read first (in this order)

1. [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md): how everything fits. §12 is the recipe.
2. [internal/setup/](internal/setup/): the reference feature. Copy its shape.
3. Your work package in [docs/WORK_PACKAGES.md](docs/WORK_PACKAGES.md).
4. The parts of the specs your package touches:
   - [api/contract.yaml](api/contract.yaml): exact operations and schemas (**do not change it**);
   - `../docs/design/permissions.md`: who may do what, redaction, the test matrix;
   - [db/migrations/](db/migrations/) and `../docs/design/data-model.md`: tables and triggers;
   - `../mocks/handlers.ts`: a working mock of the API's behavior (useful for the canvas view).
   (`../` is the parent Netopology repo; this repo is its `backend/` submodule.)
5. [docs/TESTING.md](docs/TESTING.md) and [docs/ENVIRONMENT.md](docs/ENVIRONMENT.md).

## Workflow

```sh
go tool task setup && go tool task db:up     # once
go tool task dev                              # run the api
go tool task gen                              # after changing SQL or operations
go tool task check                            # before every commit: gen + lint + all tests
```

1. Create a branch `wp-NN-short-name` from `main`.
2. SQL first (`db/queries/<feature>.sql`, `task gen:sqlc`), then service, then handler, then tests.
3. Register operations with `httpx.Op` using the **exact** values from the contract. The contract test
   (`internal/server/contract_test.go`) fails on any mismatch; read its message.
4. Write the integration tests listed in your package and in TESTING.md "What every work package must test".
5. `go tool task check` green. Update the State column in WORK_PACKAGES.md.
6. Commit with a conventional message (`feat(inventory): node reads with editor visibility (WP-08)`).

## Hard rules

- **Never edit** `api/contract.yaml`, merged migrations, or `internal/platform/pg/dbgen/` (generated).
  If the contract seems wrong or incomplete, stop and write the question in your PR description; do not
  invent endpoints, fields or status codes.
- Authorization in the **service**, first thing in every method. Non-members get **404**, never 403.
- Workspace content only inside `db.InWorkspace`, and every SQL statement filters by `workspace_id`.
- Mutation + `auditlog.Write` + `events.Emit` in **one** transaction.
- Redact on every read path; never return sensitive values for the frontend to hide.
- Feature packages never import each other; use `platform/*`, `authz`, `apimodel` or a small
  consumer-side interface wired in `internal/server/server.go`.
- Services return `*apperr.Error` for expected failures; handlers return errors unchanged.
- No secrets in logs, audit metadata, errors or URLs. No `fmt.Sprintf` into SQL.
- No new dependencies without a line in your PR explaining why the standard library or existing ones
  do not suffice. Dev tools go in `tools/go.mod`.
- Don't mock the database. Integration tests use `testutil.NewAPI`.
- Keep comments at the density of the existing code: explain *why*, reference rules
  (`// rule 5: editors can't delete what others depend on`).

## When you are done

Your PR description lists: operations implemented, rules enforced (with permissions.md references),
tests added, anything you could not do and why, and any contract question. Nothing else.
