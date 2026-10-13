-- Workspaces and members (internal/workspace, internal/member; also used by internal/setup).
-- workspaces and workspace_members have no RLS: the service checks membership first.

-- name: CreateWorkspace :one
INSERT INTO workspaces (name, slug, description, created_by)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: InsertMember :exec
INSERT INTO workspace_members (workspace_id, user_id, role, can_view_sensitive, added_by)
VALUES ($1, $2, $3, $4, $5);

-- TODO(WP-05): ListWorkspacesForUser (join membership), GetWorkspace, UpdateWorkspace (with
--              version check: WHERE id = $1 AND version = $2), DeleteWorkspace, SlugExists.
-- TODO(WP-06): ListMembers, GetMember, UpdateMember, DeleteMember, CountOwners (FOR UPDATE),
--              ReplaceCanvasGrants.
