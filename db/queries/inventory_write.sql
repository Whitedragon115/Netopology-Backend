-- Inventory mutations run inside pg.InWorkspace. Every statement is workspace scoped.

-- name: InsertArea :one
INSERT INTO areas (workspace_id, parent_id, name, kind, color, cidr, vlan_id, description, properties, created_by)
VALUES (sqlc.arg(workspace_id), sqlc.narg(parent_id), sqlc.arg(name), sqlc.arg(kind), sqlc.arg(color),
        sqlc.narg(cidr), sqlc.narg(vlan_id), sqlc.arg(description), sqlc.arg(properties), sqlc.arg(created_by))
RETURNING *;

-- name: LockArea :one
SELECT * FROM areas WHERE workspace_id = $1 AND id = $2 FOR UPDATE;

-- name: UpdateAreaInventory :one
UPDATE areas SET parent_id = sqlc.narg(parent_id), name = sqlc.arg(name), kind = sqlc.arg(kind),
  color = sqlc.arg(color), cidr = sqlc.narg(cidr), vlan_id = sqlc.narg(vlan_id),
  description = sqlc.arg(description), properties = sqlc.arg(properties)
WHERE workspace_id = sqlc.arg(workspace_id) AND id = sqlc.arg(id) AND version = sqlc.arg(version)
RETURNING *;

-- name: DeleteAreaInventory :execrows
DELETE FROM areas WHERE workspace_id = $1 AND id = $2 AND version = $3;

-- name: InsertNode :one
INSERT INTO nodes (workspace_id, area_id, slug, name, category, type, icon, description, properties, is_protected, created_by)
VALUES (sqlc.arg(workspace_id), sqlc.narg(area_id), sqlc.arg(slug), sqlc.arg(name), sqlc.arg(category),
        sqlc.arg(type), sqlc.narg(icon), sqlc.arg(description), sqlc.arg(properties), sqlc.arg(is_protected), sqlc.arg(created_by))
RETURNING *;

-- name: LockNode :one
SELECT * FROM nodes WHERE workspace_id = $1 AND id = $2 FOR UPDATE;

-- name: UpdateNodeInventory :one
UPDATE nodes SET area_id = sqlc.narg(area_id), slug = sqlc.arg(slug), name = sqlc.arg(name),
  category = sqlc.arg(category), type = sqlc.arg(type), icon = sqlc.narg(icon),
  description = sqlc.arg(description), properties = sqlc.arg(properties), is_protected = sqlc.arg(is_protected)
WHERE workspace_id = sqlc.arg(workspace_id) AND id = sqlc.arg(id) AND version = sqlc.arg(version)
RETURNING *;

-- name: SetNodeHealthInventory :one
UPDATE nodes SET health_status = sqlc.arg(status), health_source = 'manual',
  health_message = sqlc.narg(message), health_checked_at = now()
WHERE workspace_id = sqlc.arg(workspace_id) AND id = sqlc.arg(id)
RETURNING *;

-- name: DeleteNodeInventory :execrows
DELETE FROM nodes WHERE workspace_id = $1 AND id = $2 AND version = $3;

-- name: InsertLink :one
INSERT INTO links (workspace_id, source_node_id, target_node_id, kind, directed, label,
                   source_port, target_port, properties, created_by)
VALUES (sqlc.arg(workspace_id), sqlc.arg(source_node_id), sqlc.arg(target_node_id), sqlc.arg(kind),
        sqlc.arg(directed), sqlc.arg(label), sqlc.narg(source_port), sqlc.narg(target_port),
        sqlc.arg(properties), sqlc.arg(created_by))
RETURNING *;

-- name: LockLink :one
SELECT * FROM links WHERE workspace_id = $1 AND id = $2 FOR UPDATE;

-- name: UpdateLinkInventory :one
UPDATE links SET kind = sqlc.arg(kind), directed = sqlc.arg(directed), label = sqlc.arg(label),
  source_port = sqlc.narg(source_port), target_port = sqlc.narg(target_port), properties = sqlc.arg(properties)
WHERE workspace_id = sqlc.arg(workspace_id) AND id = sqlc.arg(id) AND version = sqlc.arg(version)
RETURNING *;

-- name: DeleteLinkInventory :execrows
DELETE FROM links WHERE workspace_id = $1 AND id = $2 AND version = $3;

-- name: InsertNote :one
INSERT INTO notes (workspace_id, node_id, link_id, area_id, canvas_id, kind, title, body,
                   is_sensitive, color, created_by, updated_by)
VALUES (sqlc.arg(workspace_id), sqlc.narg(node_id), sqlc.narg(link_id), sqlc.narg(area_id),
        sqlc.narg(canvas_id), sqlc.arg(kind), sqlc.arg(title), sqlc.arg(body),
        sqlc.arg(is_sensitive), sqlc.arg(color), sqlc.arg(created_by), sqlc.arg(updated_by))
RETURNING *;

-- name: LockNote :one
SELECT * FROM notes WHERE workspace_id = $1 AND id = $2 FOR UPDATE;

-- name: UpdateNoteInventory :one
UPDATE notes SET kind = sqlc.arg(kind), title = sqlc.arg(title), body = sqlc.arg(body),
  is_sensitive = sqlc.arg(is_sensitive), color = sqlc.arg(color), updated_by = sqlc.arg(updated_by)
WHERE workspace_id = sqlc.arg(workspace_id) AND id = sqlc.arg(id) AND version = sqlc.arg(version)
RETURNING *;

-- name: DeleteNoteInventory :execrows
DELETE FROM notes WHERE workspace_id = $1 AND id = $2 AND version = $3;

-- name: InsertInventoryPlacement :exec
INSERT INTO placements (workspace_id, canvas_id, node_id, area_id, note_id,
                        parent_placement_id, x, y, width, height, style)
VALUES (sqlc.arg(workspace_id), sqlc.arg(canvas_id), sqlc.narg(node_id), sqlc.narg(area_id), sqlc.narg(note_id),
        sqlc.narg(parent_placement_id), sqlc.arg(x), sqlc.arg(y), sqlc.narg(width), sqlc.narg(height), sqlc.arg(style));

-- name: GetInventoryCanvas :one
SELECT * FROM canvases WHERE workspace_id = $1 AND id = $2;

-- name: AreaExists :one
SELECT EXISTS(SELECT 1 FROM areas WHERE workspace_id = $1 AND id = $2);

-- name: NodeExists :one
SELECT EXISTS(SELECT 1 FROM nodes WHERE workspace_id = $1 AND id = $2);

-- name: NodeSlugExists :one
SELECT EXISTS(SELECT 1 FROM nodes WHERE workspace_id = $1 AND slug = $2);

-- name: IsGrantedTabCanvas :one
SELECT EXISTS(SELECT 1 FROM canvas_grants cg JOIN canvases c
  ON c.workspace_id = cg.workspace_id AND c.id = cg.canvas_id
  WHERE cg.workspace_id = $1 AND cg.canvas_id = $2 AND cg.user_id = $3 AND c.area_id IS NULL);

-- name: LinkExists :one
SELECT EXISTS(SELECT 1 FROM links WHERE workspace_id = $1 AND id = $2);

-- name: InventoryCanvasIDs :one
SELECT ARRAY(
  SELECT DISTINCT canvas_id FROM (
    SELECT p.canvas_id FROM placements p WHERE p.workspace_id = sqlc.arg(workspace_id) AND
      ((sqlc.arg(kind)::text = 'node' AND p.node_id = sqlc.arg(id)) OR
       (sqlc.arg(kind)::text = 'area' AND p.area_id = sqlc.arg(id)) OR
       (sqlc.arg(kind)::text = 'note' AND p.note_id = sqlc.arg(id)))
    UNION
    SELECT cv.canvas_id FROM canvas_visible_links cv WHERE cv.workspace_id = sqlc.arg(workspace_id)
      AND sqlc.arg(kind)::text = 'link' AND cv.link_id = sqlc.arg(id)
    UNION
    SELECT n.canvas_id FROM notes n WHERE n.workspace_id = sqlc.arg(workspace_id)
      AND sqlc.arg(kind)::text = 'note' AND n.id = sqlc.arg(id) AND n.canvas_id IS NOT NULL
    UNION
    SELECT p.canvas_id FROM notes n JOIN placements p ON p.workspace_id = n.workspace_id AND
      ((n.node_id IS NOT NULL AND p.node_id = n.node_id) OR
       (n.area_id IS NOT NULL AND p.area_id = n.area_id))
      WHERE n.workspace_id = sqlc.arg(workspace_id) AND sqlc.arg(kind)::text = 'note' AND n.id = sqlc.arg(id)
    UNION
    SELECT cv.canvas_id FROM notes n JOIN canvas_visible_links cv
      ON cv.workspace_id = n.workspace_id AND cv.link_id = n.link_id
      WHERE n.workspace_id = sqlc.arg(workspace_id) AND sqlc.arg(kind)::text = 'note' AND n.id = sqlc.arg(id)
  ) displayed
)::uuid[] AS canvas_ids;

-- name: InventoryUsedOutsideGrants :one
SELECT EXISTS(
  SELECT 1 FROM placements p WHERE p.workspace_id = sqlc.arg(workspace_id)
    AND ((sqlc.arg(kind)::text = 'node' AND p.node_id = sqlc.arg(id)) OR
         (sqlc.arg(kind)::text = 'area' AND p.area_id = sqlc.arg(id)) OR
         (sqlc.arg(kind)::text = 'note' AND p.note_id = sqlc.arg(id)))
    AND NOT EXISTS (SELECT 1 FROM canvas_grants cg WHERE cg.workspace_id = p.workspace_id
      AND cg.canvas_id = p.canvas_id AND cg.user_id = sqlc.arg(user_id))
);
