# Implementation decisions — 2026-10-14

- User authorized completion of remaining jobs without questions; record decisions here.
- Work on `alpha`, overriding the per-WP branch instruction in AGENTS.md as explicitly requested by the user.
- The other agent owns frontend implementation. Preserve parent-repository edits and existing frontend processes; implement backend packages and coordinate through this log and the existing task board.
- Complete WP-13 first: it is independent of WP-11/WP-12 and provides an API for verifying subsequent audit writes. Then implement WP-11 and WP-12, followed by WP-14.
- Keep the immutable contract and merged migrations unchanged. If a genuine contract conflict prevents work, log it and continue independent authorized work rather than inventing API behavior.
- Use the configured PostgreSQL 18 test database through Taskfile dotenv; do not replace the remote database or reset application data.
- The prior branch consolidation is not permission to overwrite later changes made by another agent. Recheck Git state before each commit and stage only this agent's backend work.
- WP-13: audit cursors contain `(created_at, id)` so timestamp ties cannot skip rows; `action` matches an exact permission key or a prefix ending in `.`. Invalid cursors are validation failures. Reads produce no audit writes.
- WP-13 validation completed: `go tool task check` passed with PostgreSQL 18 integration tests, including pagination, all filters, admin/owner reads, editor/viewer denial and non-member hiding.
- WP-11: use Huma's streaming response callback with an explicit contract SSE string schema. The bundled SSE sender only supports integer IDs, whereas this contract requires UUID event IDs.
- WP-11: subscribe before replay; paginate replay in batches of 500. Do not discard live notifications by UUID ordering, because concurrent transactions can commit out of order. Recheck membership and canvas access for every event and heartbeat.
- WP-11: deleted resources cannot be re-read. Editors receive deletion IDs only if the outbox records a canvas they currently can read; otherwise omit them. Empty canvas IDs mean workspace-wide events, as specified by the outbox API.
- WP-11 validation completed: full check passed with PostgreSQL 18 integration tests for commit versus rollback notifications, viewer redaction, editor visibility, canvas access filtering and Last-Event-ID replay.
