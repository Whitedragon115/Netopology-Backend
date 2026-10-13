-- Workspaces and members (internal/workspace, internal/member; also used by internal/setup).
-- workspaces and workspace_members have no RLS: the service checks membership first.

-- name: CreateWorkspace :one
INSERT INTO workspaces (name, slug, description, created_by)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: InsertMember :exec
INSERT INTO workspace_members (workspace_id, user_id, role, can_view_sensitive, added_by)
VALUES ($1, $2, $3, $4, $5);

-- name: ListWorkspacesForUser :many
SELECT w.*, m.role AS member_role, m.can_view_sensitive
FROM workspaces w
JOIN workspace_members m ON m.workspace_id = w.id
WHERE m.user_id = $1
ORDER BY w.created_at, w.id;

-- name: GetWorkspace :one
SELECT * FROM workspaces WHERE id = $1;

-- name: LockWorkspace :one
SELECT * FROM workspaces WHERE id = $1 FOR UPDATE;

-- name: UpdateWorkspace :one
UPDATE workspaces
SET name = $2, slug = $3, description = $4, is_protected = $5, settings = $6
WHERE id = $1 AND version = $7
RETURNING *;

-- name: DeleteWorkspace :execrows
DELETE FROM workspaces WHERE id = $1 AND version = $2;

-- name: SlugExists :one
SELECT EXISTS(SELECT 1 FROM workspaces WHERE slug = $1);
-- TODO(WP-06): ListMembers, GetMember, UpdateMember, DeleteMember, CountOwners (FOR UPDATE),
--              ReplaceCanvasGrants.
