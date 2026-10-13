-- name: ListAuditEntries :many
SELECT a.*, u.display_name AS actor_display_name
FROM audit_log a
LEFT JOIN users u ON a.actor_type = 'user' AND u.id = a.actor_id
WHERE a.workspace_id = sqlc.arg(workspace_id)
  AND (sqlc.narg(actor_id)::uuid IS NULL OR a.actor_id = sqlc.narg(actor_id))
  AND (sqlc.narg(target_id)::uuid IS NULL OR a.target_id = sqlc.narg(target_id))
  AND (sqlc.narg(since)::timestamptz IS NULL OR a.created_at >= sqlc.narg(since))
  AND (sqlc.arg(action)::text = '' OR a.action = sqlc.arg(action)
       OR (right(sqlc.arg(action), 1) = '.' AND starts_with(a.action, sqlc.arg(action))))
  AND (sqlc.narg(before_time)::timestamptz IS NULL
       OR (a.created_at, a.id) < (sqlc.narg(before_time), sqlc.narg(before_id)::uuid))
ORDER BY a.created_at DESC, a.id DESC
LIMIT sqlc.arg(page_limit);
