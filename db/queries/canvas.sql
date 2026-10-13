-- All canvas data queries run inside pg.InWorkspace and filter workspace_id.

-- name: ListTabCanvases :many
SELECT c.* FROM canvases c WHERE c.workspace_id = sqlc.arg(workspace_id) AND c.area_id IS NULL
  AND (sqlc.narg(editor_id)::uuid IS NULL OR EXISTS (
    SELECT 1 FROM canvas_grants cg WHERE cg.workspace_id=c.workspace_id AND cg.canvas_id=c.id
      AND cg.user_id=sqlc.narg(editor_id)::uuid))
ORDER BY c.sort_order,c.id;

-- name: GetCanvasRow :one
SELECT * FROM canvases WHERE workspace_id=$1 AND id=$2;

-- name: LockCanvasRow :one
SELECT * FROM canvases WHERE workspace_id=$1 AND id=$2 FOR UPDATE;

-- name: InsertTabCanvas :one
INSERT INTO canvases (workspace_id,name,description,sort_order,settings,is_protected,created_by)
VALUES (sqlc.arg(workspace_id),sqlc.arg(name),sqlc.arg(description),sqlc.arg(sort_order),
        sqlc.arg(settings),sqlc.arg(is_protected),sqlc.arg(created_by)) RETURNING *;

-- name: UpdateCanvasRow :one
UPDATE canvases SET name=sqlc.arg(name),description=sqlc.arg(description),sort_order=sqlc.arg(sort_order),
  viewport=sqlc.arg(viewport),settings=sqlc.arg(settings),is_protected=sqlc.arg(is_protected)
WHERE workspace_id=sqlc.arg(workspace_id) AND id=sqlc.arg(id) AND version=sqlc.arg(version) RETURNING *;

-- name: DeleteCanvasRow :execrows
DELETE FROM canvases WHERE workspace_id=$1 AND id=$2 AND version=$3;

-- name: EnsureAreaCanvas :execrows
INSERT INTO canvases (workspace_id,area_id,name,created_by)
SELECT a.workspace_id,a.id,a.name,sqlc.arg(created_by)::uuid FROM areas a
WHERE a.workspace_id=sqlc.arg(workspace_id) AND a.id=sqlc.arg(area_id)
ON CONFLICT (area_id) WHERE area_id IS NOT NULL DO NOTHING;

-- name: GetAreaCanvasRow :one
SELECT * FROM canvases WHERE workspace_id=$1 AND area_id=$2;

-- name: ListAreaBreadcrumb :many
WITH RECURSIVE chain AS (
  SELECT a.id, a.parent_id, a.name, 0 AS depth FROM areas a
  WHERE a.workspace_id=sqlc.arg(workspace_id) AND a.id=sqlc.arg(area_id)
  UNION ALL
  SELECT parent.id, parent.parent_id, parent.name, chain.depth+1 FROM areas parent
  JOIN chain ON chain.parent_id=parent.id
  WHERE parent.workspace_id=sqlc.arg(workspace_id)
)
SELECT chain.id,chain.name,c.id AS canvas_id FROM chain
LEFT JOIN canvases c ON c.workspace_id=sqlc.arg(workspace_id) AND c.area_id=chain.id
ORDER BY chain.depth DESC;

-- name: FindAreaStartingTab :one
SELECT c.id,c.name FROM placements p
JOIN canvases c ON c.workspace_id=p.workspace_id AND c.id=p.canvas_id AND c.area_id IS NULL
WHERE p.workspace_id=sqlc.arg(workspace_id) AND p.area_id=sqlc.arg(area_id)
ORDER BY c.sort_order,c.id LIMIT 1;

-- name: ListCanvasPlacements :many
SELECT * FROM placements WHERE workspace_id=$1 AND canvas_id=$2 ORDER BY z_index,id;

-- name: ListCanvasShapes :many
SELECT * FROM canvas_shapes WHERE workspace_id=$1 AND canvas_id=$2 ORDER BY z_index,id;

-- name: ListCanvasLinkOverrides :many
SELECT * FROM placements WHERE workspace_id=$1 AND canvas_id=$2 AND link_id IS NOT NULL ORDER BY id;

-- name: ListCanvasViewLinkIDs :many
SELECT l.id FROM links l JOIN canvases c ON c.workspace_id=l.workspace_id AND c.id=sqlc.arg(canvas_id)
WHERE l.workspace_id=sqlc.arg(workspace_id)
  AND NOT (coalesce(c.settings -> 'hiddenLinkKinds','[]'::jsonb) ? l.kind)
  AND EXISTS (SELECT 1 FROM placements p WHERE p.workspace_id=l.workspace_id AND p.canvas_id=c.id AND p.node_id=l.source_node_id)
  AND (c.area_id IS NOT NULL OR EXISTS (SELECT 1 FROM placements p WHERE p.workspace_id=l.workspace_id AND p.canvas_id=c.id AND p.node_id=l.target_node_id))
  AND NOT EXISTS (SELECT 1 FROM placements p WHERE p.workspace_id=l.workspace_id AND p.canvas_id=c.id AND p.link_id=l.id AND p.hidden)
UNION
SELECT l.id FROM links l JOIN canvases c ON c.workspace_id=l.workspace_id AND c.id=sqlc.arg(canvas_id)
WHERE l.workspace_id=sqlc.arg(workspace_id) AND c.area_id IS NOT NULL
  AND NOT (coalesce(c.settings -> 'hiddenLinkKinds','[]'::jsonb) ? l.kind)
  AND EXISTS (SELECT 1 FROM placements p WHERE p.workspace_id=l.workspace_id AND p.canvas_id=c.id AND p.node_id=l.target_node_id)
  AND NOT EXISTS (SELECT 1 FROM placements p WHERE p.workspace_id=l.workspace_id AND p.canvas_id=c.id AND p.link_id=l.id AND p.hidden)
ORDER BY id;

-- name: ListAreaChildNodes :many
SELECT id FROM nodes WHERE workspace_id=$1 AND area_id=$2 ORDER BY id;

-- name: ListAreaChildAreas :many
SELECT id FROM areas WHERE workspace_id=$1 AND parent_id=$2 ORDER BY id;

-- name: SeedAreaCanvasNodes :exec
INSERT INTO placements (workspace_id,canvas_id,node_id,x,y)
SELECT n.workspace_id,sqlc.arg(canvas_id),n.id,((row_number() OVER (ORDER BY n.id)-1)::integer % 5)*180,
  ((row_number() OVER (ORDER BY n.id)-1)::integer / 5)*120
FROM nodes n WHERE n.workspace_id=sqlc.arg(workspace_id) AND n.area_id=sqlc.arg(area_id)
ON CONFLICT (canvas_id,node_id) WHERE node_id IS NOT NULL DO NOTHING;

-- name: SeedAreaCanvasAreas :exec
INSERT INTO placements (workspace_id,canvas_id,area_id,x,y,width,height,collapsed)
SELECT a.workspace_id,sqlc.arg(canvas_id),a.id,((row_number() OVER (ORDER BY a.id)-1)::integer % 3)*320,
  ((row_number() OVER (ORDER BY a.id)-1)::integer / 3)*260,280,210,true
FROM areas a WHERE a.workspace_id=sqlc.arg(workspace_id) AND a.parent_id=sqlc.arg(area_id)
ON CONFLICT (canvas_id,area_id) WHERE area_id IS NOT NULL DO NOTHING;

-- name: InsertCanvasPlacement :one
INSERT INTO placements (workspace_id,canvas_id,node_id,area_id,note_id,parent_placement_id,x,y,width,height,z_index,collapsed,style)
VALUES (sqlc.arg(workspace_id),sqlc.arg(canvas_id),sqlc.narg(node_id),sqlc.narg(area_id),sqlc.narg(note_id),
        sqlc.narg(parent_placement_id),sqlc.arg(x),sqlc.arg(y),sqlc.narg(width),sqlc.narg(height),
        sqlc.arg(z_index),sqlc.arg(collapsed),sqlc.arg(style)) RETURNING id;

-- name: GetCanvasPlacement :one
SELECT * FROM placements WHERE workspace_id=$1 AND canvas_id=$2 AND id=$3 FOR UPDATE;

-- name: UpdateCanvasPlacement :one
UPDATE placements SET parent_placement_id=sqlc.narg(parent_placement_id),x=sqlc.arg(x),y=sqlc.arg(y),
  width=sqlc.narg(width),height=sqlc.narg(height),z_index=sqlc.arg(z_index),collapsed=sqlc.arg(collapsed),style=sqlc.arg(style)
WHERE workspace_id=sqlc.arg(workspace_id) AND canvas_id=sqlc.arg(canvas_id) AND id=sqlc.arg(id) RETURNING id;

-- name: DeleteCanvasPlacement :execrows
DELETE FROM placements WHERE workspace_id=$1 AND canvas_id=$2 AND id=$3 AND link_id IS NULL;

-- name: ReparentCanvasNode :exec
UPDATE nodes SET area_id=sqlc.narg(area_id) WHERE workspace_id=sqlc.arg(workspace_id) AND id=sqlc.arg(node_id);

-- name: GetCanvasParentArea :one
SELECT area_id FROM placements WHERE workspace_id=$1 AND canvas_id=$2 AND id=$3 AND area_id IS NOT NULL;

-- name: AdoptAreaNodes :exec
UPDATE placements p SET parent_placement_id=sqlc.arg(area_placement_id)
FROM nodes n WHERE p.workspace_id=sqlc.arg(workspace_id) AND p.canvas_id=sqlc.arg(canvas_id)
  AND p.node_id=n.id AND n.workspace_id=p.workspace_id AND n.area_id=sqlc.arg(area_id)
  AND p.parent_placement_id IS NULL;

-- name: DeleteOrphanLinkOverrides :many
DELETE FROM placements p WHERE p.workspace_id=sqlc.arg(workspace_id) AND p.canvas_id=sqlc.arg(canvas_id)
  AND p.link_id IS NOT NULL AND NOT EXISTS (
    SELECT 1 FROM links l JOIN placements ps ON ps.workspace_id=l.workspace_id AND ps.canvas_id=p.canvas_id AND ps.node_id=l.source_node_id
      JOIN placements pt ON pt.workspace_id=l.workspace_id AND pt.canvas_id=p.canvas_id AND pt.node_id=l.target_node_id
    WHERE l.workspace_id=p.workspace_id AND l.id=p.link_id
  ) RETURNING p.id;

-- name: InsertCanvasShape :one
INSERT INTO canvas_shapes (workspace_id,canvas_id,kind,x,y,width,height,rotation,z_index,style,data,created_by)
VALUES (sqlc.arg(workspace_id),sqlc.arg(canvas_id),sqlc.arg(kind),sqlc.arg(x),sqlc.arg(y),
        sqlc.narg(width),sqlc.narg(height),sqlc.arg(rotation),sqlc.arg(z_index),sqlc.arg(style),
        sqlc.arg(data),sqlc.arg(created_by)) RETURNING id;

-- name: GetCanvasShape :one
SELECT * FROM canvas_shapes WHERE workspace_id=$1 AND canvas_id=$2 AND id=$3 FOR UPDATE;

-- name: UpdateCanvasShape :one
UPDATE canvas_shapes SET x=sqlc.arg(x),y=sqlc.arg(y),width=sqlc.narg(width),height=sqlc.narg(height),
  rotation=sqlc.arg(rotation),z_index=sqlc.arg(z_index),style=sqlc.arg(style),data=sqlc.arg(data)
WHERE workspace_id=sqlc.arg(workspace_id) AND canvas_id=sqlc.arg(canvas_id) AND id=sqlc.arg(id) RETURNING id;

-- name: DeleteCanvasShape :execrows
DELETE FROM canvas_shapes WHERE workspace_id=$1 AND canvas_id=$2 AND id=$3;

-- name: GetCanvasLinkOverride :one
SELECT * FROM placements WHERE workspace_id=$1 AND canvas_id=$2 AND link_id=$3 FOR UPDATE;

-- name: UpsertCanvasLinkOverride :exec
INSERT INTO placements (workspace_id,canvas_id,link_id,hidden,style,waypoints)
VALUES (sqlc.arg(workspace_id),sqlc.arg(canvas_id),sqlc.arg(link_id),sqlc.arg(hidden),sqlc.arg(style),sqlc.narg(waypoints))
ON CONFLICT (canvas_id,link_id) WHERE link_id IS NOT NULL
DO UPDATE SET hidden=EXCLUDED.hidden,style=EXCLUDED.style,waypoints=EXCLUDED.waypoints;

-- name: DeleteCanvasLinkOverride :execrows
DELETE FROM placements WHERE workspace_id=$1 AND canvas_id=$2 AND link_id=$3;

-- name: CanvasHasNode :one
SELECT EXISTS(SELECT 1 FROM placements WHERE workspace_id=$1 AND canvas_id=$2 AND node_id=$3);

-- name: CanvasItemExists :one
SELECT EXISTS(
  SELECT 1 FROM nodes n WHERE n.workspace_id=sqlc.arg(workspace_id) AND n.id=sqlc.arg(id) AND sqlc.arg(kind)::text='node'
  UNION ALL SELECT 1 FROM areas a WHERE a.workspace_id=sqlc.arg(workspace_id) AND a.id=sqlc.arg(id) AND sqlc.arg(kind)::text='area'
  UNION ALL SELECT 1 FROM notes n WHERE n.workspace_id=sqlc.arg(workspace_id) AND n.id=sqlc.arg(id) AND sqlc.arg(kind)::text='note'
);

-- name: CanvasLinkEndpoints :one
SELECT source_node_id,target_node_id FROM links WHERE workspace_id=$1 AND id=$2;

-- name: SetCanvasLock :one
UPDATE canvases SET locked_by=sqlc.narg(locked_by),locked_at=CASE WHEN sqlc.narg(locked_by)::uuid IS NULL THEN NULL ELSE now() END
WHERE workspace_id=sqlc.arg(workspace_id) AND id=sqlc.arg(id) RETURNING *;
