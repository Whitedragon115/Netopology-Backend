-- Authorization lookups (internal/authz). Used by every workspace-scoped request.

-- name: GetMembership :one
-- workspace_members has no RLS (it decides access). No row = not a member = 404.
SELECT m.role, m.can_view_sensitive, w.is_protected AS workspace_protected
FROM workspace_members m
JOIN workspaces w ON w.id = m.workspace_id
WHERE m.workspace_id = $1 AND m.user_id = $2;

-- name: ListGrantedCanvasIDs :many
-- Editors only. canvas_grants has RLS: call inside pg.InWorkspace.
SELECT canvas_id FROM canvas_grants WHERE workspace_id = $1 AND user_id = $2;
