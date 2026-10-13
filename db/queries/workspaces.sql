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
-- Membership records are access-control data, so these joins run outside RLS.
-- name: ListMembers :many
SELECT m.workspace_id, m.user_id, m.role, m.can_view_sensitive, m.created_at,
       u.username, u.display_name
FROM workspace_members m
JOIN users u ON u.id = m.user_id
WHERE m.workspace_id = $1
ORDER BY m.created_at, m.user_id;

-- name: GetMember :one
SELECT m.workspace_id, m.user_id, m.role, m.can_view_sensitive, m.created_at,
       u.username, u.display_name
FROM workspace_members m
JOIN users u ON u.id = m.user_id
WHERE m.workspace_id = $1 AND m.user_id = $2;

-- name: LockMember :one
SELECT * FROM workspace_members
WHERE workspace_id = $1 AND user_id = $2 FOR UPDATE;

-- name: LockOwners :many
SELECT user_id FROM workspace_members
WHERE workspace_id = $1 AND role = 'owner' FOR UPDATE;

-- name: UpdateMember :execrows
UPDATE workspace_members SET role = $3, can_view_sensitive = $4
WHERE workspace_id = $1 AND user_id = $2;

-- name: DeleteMember :execrows
DELETE FROM workspace_members WHERE workspace_id = $1 AND user_id = $2;

-- name: CountTabCanvases :one
SELECT count(*) FROM canvases
WHERE workspace_id = $1 AND area_id IS NULL AND id = ANY($2::uuid[]);

-- name: ListMemberGrants :many
SELECT canvas_id FROM canvas_grants
WHERE workspace_id = $1 AND user_id = $2 ORDER BY canvas_id;

-- name: ReplaceCanvasGrants :exec
WITH desired AS (SELECT unnest($3::uuid[]) AS canvas_id),
deleted AS (
  DELETE FROM canvas_grants
  WHERE workspace_id = $1 AND user_id = $2 AND canvas_id NOT IN (SELECT canvas_id FROM desired)
  RETURNING canvas_id
)
INSERT INTO canvas_grants (workspace_id, user_id, canvas_id, granted_by)
SELECT $1, $2, desired.canvas_id, $4 FROM desired CROSS JOIN (SELECT count(*) FROM deleted) AS guard
ON CONFLICT (canvas_id, user_id) DO NOTHING;
