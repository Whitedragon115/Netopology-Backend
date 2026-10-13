# Testing

Three kinds of tests, all run by `go tool task test` (and CI):

| Kind | Where | Needs |
|---|---|---|
| Unit | next to the code (`*_test.go`), pure functions: redaction, policy helpers, slug, crypto | nothing |
| Contract | `internal/server/contract_test.go`: generated OpenAPI vs `api/contract.yaml`, permission keys, event types | nothing |
| Integration | `<feature>/*_test.go` (package `<feature>_test`), black-box through HTTP | Postgres 18 superuser URL |

## Integration tests

```go
func TestSomething(t *testing.T) {
	api := testutil.NewAPI(t)   // fresh migrated database + the real HTTP stack (all middleware)
	owner := api.Owner()        // completes setup: server owner + workspace "Home lab"
	res := owner.Do(http.MethodGet, "/workspaces", nil).Expect(t, http.StatusOK)
	var page struct{ Items []struct{ ID, Slug string } }
	res.JSON(t, &page)
}
```

- `testutil.Postgres(t)` creates a database **per test** from a migrated template (fast) and connects
  as a non-superuser role, so **RLS is active** like in production. It is dropped after the test.
- `NETOPOLOGY_TEST_ADMIN_URL` must point at a Postgres **18** superuser. With `task db:up` that is
  `postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable` (already in `.env.example`).
  Without it, integration tests are **skipped**, not failed; CI always sets it.
- Clients keep cookies and send the right `Origin`, like a browser. Each `api.Client()` is a new browser.
- Extend `testutil` with fixture helpers as features land (WP-03 adds `Login`, WP-04 `CreateUser`,
  WP-07 `AddMember`/`ClientAs(role)`, WP-09 `SeedInventory`). Fixtures go **through the API** when the
  endpoint exists; use `api.DB.OwnerURL` for raw SQL only when it does not exist yet.
- Tests must not depend on each other or on order; each has its own database.

## What every work package must test

1. Happy path of every operation, asserting the response shape (decode into a struct).
2. Every service rule it implements (one sub-test per rule, named after the rule).
3. Its rows of the permission matrix (permissions.md §5), in `<feature>/permissions_test.go`.
   Use a table: `{role, request, wantStatus, wantCode}`.
4. Redaction for every read path that returns sensitive data (assert the *absence* of values).
5. Audit: one entry per mutation with the right action (query `audit_log` via `api.DB.OwnerURL` until WP-13 lands).

## The schema checks

`db/tests/scenarios.sql` runs 27 checks of triggers/constraints/RLS directly in SQL as the app role:
`task db:scenarios`. Add a check whenever a migration adds a trigger or constraint.

## Contract strictness

While features are being built, unimplemented operations are only *logged* by the contract test.
`task test:strict` fails on them; it becomes the CI default after WP-14.
