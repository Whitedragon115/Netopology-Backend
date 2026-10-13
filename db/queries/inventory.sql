-- Every query here runs inside pg.InWorkspace and filters by workspace_id.
-- Editor visibility is computed in SQL, not by loading the full inventory into Go.

-- name: VisibleInventoryIDs :one
WITH RECURSIVE visible_canvases(id) AS (
  SELECT cg.canvas_id FROM canvas_grants cg
  JOIN canvases c ON c.workspace_id = cg.workspace_id AND c.id = cg.canvas_id
  WHERE cg.workspace_id = $1 AND cg.user_id = $2 AND c.area_id IS NULL
  UNION
  SELECT inner_canvas.id FROM visible_canvases vc
  JOIN placements p ON p.workspace_id = $1 AND p.canvas_id = vc.id AND p.area_id IS NOT NULL
  JOIN canvases inner_canvas ON inner_canvas.workspace_id = $1 AND inner_canvas.area_id = p.area_id
),
visible_nodes AS (
  SELECT DISTINCT p.node_id AS id FROM placements p
  JOIN visible_canvases vc ON vc.id = p.canvas_id
  WHERE p.workspace_id = $1 AND p.node_id IS NOT NULL
),
visible_areas AS (
  SELECT DISTINCT p.area_id AS id FROM placements p
  JOIN visible_canvases vc ON vc.id = p.canvas_id
  WHERE p.workspace_id = $1 AND p.area_id IS NOT NULL
  UNION
  SELECT n.area_id FROM nodes n JOIN visible_nodes vn ON vn.id = n.id
  WHERE n.workspace_id = $1 AND n.area_id IS NOT NULL
),
visible_links AS (
  SELECT l.id FROM links l
  JOIN visible_nodes source ON source.id = l.source_node_id
  JOIN visible_nodes target ON target.id = l.target_node_id
  WHERE l.workspace_id = $1
),
visible_notes AS (
  SELECT n.id FROM notes n WHERE n.workspace_id = $1 AND (
    (n.node_id IS NULL AND n.link_id IS NULL AND n.area_id IS NULL AND n.canvas_id IS NULL)
    OR n.node_id IN (SELECT id FROM visible_nodes)
    OR n.link_id IN (SELECT id FROM visible_links)
    OR n.area_id IN (SELECT id FROM visible_areas)
    OR n.canvas_id IN (SELECT id FROM visible_canvases)
    OR EXISTS (SELECT 1 FROM placements p JOIN visible_canvases vc ON vc.id = p.canvas_id
               WHERE p.workspace_id = $1 AND p.note_id = n.id)
  )
)
SELECT
  ARRAY(SELECT id FROM visible_canvases)::uuid[] AS canvas_ids,
  ARRAY(SELECT id FROM visible_nodes)::uuid[] AS node_ids,
  ARRAY(SELECT id FROM visible_areas)::uuid[] AS area_ids,
  ARRAY(SELECT id FROM visible_links)::uuid[] AS link_ids,
  ARRAY(SELECT id FROM visible_notes)::uuid[] AS note_ids;

-- name: ListAreasVisible :many
SELECT a.*, (SELECT count(*) FROM nodes n WHERE n.workspace_id = a.workspace_id AND n.area_id = a.id
              AND (sqlc.arg(all_visible)::boolean OR n.id = ANY(sqlc.arg(visible_node_ids)::uuid[]))) AS node_count
FROM areas a
WHERE a.workspace_id = sqlc.arg(workspace_id)
  AND (sqlc.arg(all_visible)::boolean OR a.id = ANY(sqlc.arg(visible_ids)::uuid[]))
  AND (NOT sqlc.arg(filter_parent)::boolean OR
       (sqlc.arg(root_only)::boolean AND a.parent_id IS NULL) OR
       a.parent_id = sqlc.narg(parent_id)::uuid)
  AND (sqlc.narg(after_id)::uuid IS NULL OR a.id > sqlc.narg(after_id)::uuid)
ORDER BY a.id LIMIT sqlc.arg(page_limit);

-- name: GetAreaVisible :one
SELECT a.*, (SELECT count(*) FROM nodes n WHERE n.workspace_id = a.workspace_id AND n.area_id = a.id
              AND (sqlc.arg(all_visible)::boolean OR n.id = ANY(sqlc.arg(visible_node_ids)::uuid[]))) AS node_count
FROM areas a
WHERE a.workspace_id = sqlc.arg(workspace_id) AND a.id = sqlc.arg(id)
  AND (sqlc.arg(all_visible)::boolean OR a.id = ANY(sqlc.arg(visible_ids)::uuid[]));

-- name: ListNodesVisible :many
SELECT n.* FROM nodes n
WHERE n.workspace_id = sqlc.arg(workspace_id)
  AND (sqlc.arg(all_visible)::boolean OR n.id = ANY(sqlc.arg(visible_ids)::uuid[]))
  AND (sqlc.narg(area_id)::uuid IS NULL OR n.area_id = sqlc.narg(area_id)::uuid)
  AND (sqlc.narg(category)::text IS NULL OR n.category = sqlc.narg(category)::text)
  AND (sqlc.narg(canvas_id)::uuid IS NULL OR EXISTS (
       SELECT 1 FROM placements p WHERE p.workspace_id = n.workspace_id
       AND p.node_id = n.id AND p.canvas_id = sqlc.narg(canvas_id)::uuid))
  AND (sqlc.narg(health)::text IS NULL OR n.health_status = sqlc.narg(health)::text)
  AND (sqlc.narg(search)::text IS NULL OR n.name ILIKE '%' || sqlc.narg(search)::text || '%'
       OR n.slug ILIKE '%' || sqlc.narg(search)::text || '%'
       OR EXISTS (SELECT 1 FROM jsonb_array_elements(n.properties) f
                  WHERE NOT COALESCE((f->>'sensitive')::boolean, false)
                  AND f->>'value' ILIKE '%' || sqlc.narg(search)::text || '%'))
  AND (sqlc.narg(after_id)::uuid IS NULL OR n.id > sqlc.narg(after_id)::uuid)
ORDER BY n.id LIMIT sqlc.arg(page_limit);

-- name: GetNodeVisible :one
SELECT * FROM nodes
WHERE workspace_id = sqlc.arg(workspace_id) AND id = sqlc.arg(id)
  AND (sqlc.arg(all_visible)::boolean OR id = ANY(sqlc.arg(visible_ids)::uuid[]));

-- name: ListLinksVisible :many
SELECT * FROM links
WHERE workspace_id = sqlc.arg(workspace_id)
  AND (sqlc.arg(all_visible)::boolean OR id = ANY(sqlc.arg(visible_ids)::uuid[]))
  AND (sqlc.narg(node_id)::uuid IS NULL OR source_node_id = sqlc.narg(node_id)::uuid OR target_node_id = sqlc.narg(node_id)::uuid)
  AND (sqlc.narg(kind)::text IS NULL OR kind = sqlc.narg(kind)::text)
  AND (sqlc.narg(after_id)::uuid IS NULL OR id > sqlc.narg(after_id)::uuid)
ORDER BY id LIMIT sqlc.arg(page_limit);

-- name: GetLinkVisible :one
SELECT * FROM links
WHERE workspace_id = sqlc.arg(workspace_id) AND id = sqlc.arg(id)
  AND (sqlc.arg(all_visible)::boolean OR id = ANY(sqlc.arg(visible_ids)::uuid[]));

-- name: ListNotesVisible :many
SELECT * FROM notes
WHERE workspace_id = sqlc.arg(workspace_id)
  AND (sqlc.arg(all_visible)::boolean OR id = ANY(sqlc.arg(visible_ids)::uuid[]))
  AND ((sqlc.arg(target_type)::text = 'workspace' AND node_id IS NULL AND link_id IS NULL AND area_id IS NULL AND canvas_id IS NULL)
    OR (sqlc.arg(target_type)::text = 'node' AND node_id = sqlc.narg(target_id)::uuid)
    OR (sqlc.arg(target_type)::text = 'link' AND link_id = sqlc.narg(target_id)::uuid)
    OR (sqlc.arg(target_type)::text = 'area' AND area_id = sqlc.narg(target_id)::uuid)
    OR (sqlc.arg(target_type)::text = 'canvas' AND canvas_id = sqlc.narg(target_id)::uuid))
  AND (sqlc.narg(kind)::text IS NULL OR kind = sqlc.narg(kind)::text)
  AND (sqlc.narg(after_id)::uuid IS NULL OR id > sqlc.narg(after_id)::uuid)
ORDER BY id LIMIT sqlc.arg(page_limit);

-- name: GetNoteVisible :one
SELECT * FROM notes
WHERE workspace_id = sqlc.arg(workspace_id) AND id = sqlc.arg(id)
  AND (sqlc.arg(all_visible)::boolean OR id = ANY(sqlc.arg(visible_ids)::uuid[]));

-- name: ListNodeUsageCanvases :many
SELECT c.id, c.name FROM placements p
JOIN canvases c ON c.workspace_id = p.workspace_id AND c.id = p.canvas_id
WHERE p.workspace_id = $1 AND p.node_id = $2 ORDER BY c.id;

-- name: CountNodeLinks :one
SELECT count(*) FROM links WHERE workspace_id = sqlc.arg(workspace_id)
  AND (source_node_id = sqlc.arg(node_id) OR target_node_id = sqlc.arg(node_id))
  AND (sqlc.arg(all_visible)::boolean OR id = ANY(sqlc.arg(visible_ids)::uuid[]));

-- name: CountNodeNotes :one
SELECT count(*) FROM notes WHERE workspace_id = sqlc.arg(workspace_id) AND node_id = sqlc.arg(node_id)
  AND (sqlc.arg(all_visible)::boolean OR id = ANY(sqlc.arg(visible_ids)::uuid[]));
