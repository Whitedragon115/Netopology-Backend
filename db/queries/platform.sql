-- Cross-cutting writes done in the same transaction as the change they describe.

-- name: InsertAuditEntry :exec
-- audit_log is append-only (a trigger blocks UPDATE). action = the permission key.
INSERT INTO audit_log (workspace_id, actor_type, actor_id, action, target_type, target_id, metadata, ip, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: InsertWorkspaceEvent :one
-- Must run inside pg.InWorkspace (RLS). Notifies listeners on commit.
INSERT INTO workspace_events (workspace_id, type, subject_id, canvas_ids, actor_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: GetWorkspaceEvent :one
SELECT * FROM workspace_events WHERE workspace_id = $1 AND id = $2;

-- name: ListWorkspaceEventsAfter :many
-- Resume after Last-Event-ID (UUIDv7 ids sort by time).
SELECT * FROM workspace_events
WHERE workspace_id = $1 AND id > $2
ORDER BY id
LIMIT 500;

-- name: PruneWorkspaceEvents :one
SELECT prune_workspace_events(make_interval(hours => sqlc.arg(keep_hours)::int))::bigint AS deleted;
