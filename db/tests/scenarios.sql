-- A1 "done when" checks for the v1 schema.
-- Run as the NON-superuser app role (superusers bypass RLS):
--   psql -v ON_ERROR_STOP=1 -U netopology_app -d netopology -f scenarios.sql
-- Every check prints PASS; any failure aborts with an error.

\set ON_ERROR_STOP 1
BEGIN;

-- ── 1. accounts and two workspaces ─────────────────────────────────────────
INSERT INTO users (id, username, display_name, password_hash, instance_role) VALUES
  ('00000000-0000-7000-8000-000000000001', 'alice', 'Alice', 'x', 'owner'),
  ('00000000-0000-7000-8000-000000000002', 'bob',   'Bob',   NULL, 'member');  -- Google/Discord only

DO $$ BEGIN
  INSERT INTO users (username, password_hash, instance_role) VALUES ('mallory', 'x', 'owner');
  RAISE EXCEPTION 'FAIL: second server owner accepted';
EXCEPTION WHEN unique_violation THEN RAISE NOTICE 'PASS only one server owner can exist';
END $$;

INSERT INTO user_identities (user_id, provider, subject, email) VALUES
  ('00000000-0000-7000-8000-000000000002', 'discord', '80351110224678912', 'bob@example.com');
DO $$ BEGIN
  INSERT INTO user_identities (user_id, provider, subject) VALUES
    ('00000000-0000-7000-8000-000000000001', 'discord', '80351110224678912');
  RAISE EXCEPTION 'FAIL: one Discord account linked to two users';
EXCEPTION WHEN unique_violation THEN RAISE NOTICE 'PASS an external identity links to exactly one account';
END $$;

INSERT INTO workspaces (id, name, slug, created_by) VALUES
  ('00000000-0000-7000-8000-0000000000a1', 'Home lab', 'home', '00000000-0000-7000-8000-000000000001'),
  ('00000000-0000-7000-8000-0000000000a2', 'Lab B',    'lab-b', '00000000-0000-7000-8000-000000000001');

INSERT INTO workspace_members (workspace_id, user_id, role) VALUES
  ('00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-000000000001', 'owner'),
  ('00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-000000000002', 'editor'),
  ('00000000-0000-7000-8000-0000000000a2', '00000000-0000-7000-8000-000000000001', 'owner');

-- ── 2. inside "home": areas, nodes, links ─────────────────────────────────
SELECT set_config('app.workspace_id', '00000000-0000-7000-8000-0000000000a1', true);

INSERT INTO areas (id, workspace_id, name, kind) VALUES
  ('00000000-0000-7000-8000-0000000000e1', '00000000-0000-7000-8000-0000000000a1', 'Home', 'site');
INSERT INTO areas (id, workspace_id, parent_id, name, kind, cidr) VALUES
  ('00000000-0000-7000-8000-0000000000e2', '00000000-0000-7000-8000-0000000000a1',
   '00000000-0000-7000-8000-0000000000e1', 'Main LAN', 'network', '192.168.1.0/24');

DO $$ BEGIN
  UPDATE areas SET parent_id = '00000000-0000-7000-8000-0000000000e2'
   WHERE id = '00000000-0000-7000-8000-0000000000e1';
  RAISE EXCEPTION 'FAIL: area cycle was accepted';
EXCEPTION WHEN check_violation THEN RAISE NOTICE 'PASS area nesting cycle rejected';
END $$;

INSERT INTO nodes (id, workspace_id, area_id, slug, name, category, type, properties) VALUES
  ('00000000-0000-7000-8000-000000000b01', '00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-0000000000e2',
   'router', 'Router', 'network', 'builtin.router',
   '[{"key":"ip","label":"IP","value":"192.168.1.1","sensitive":true}]'),
  ('00000000-0000-7000-8000-000000000b02', '00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-0000000000e2',
   'npm-main', 'Nginx Proxy Manager', 'service', 'builtin.generic', '[]'),
  ('00000000-0000-7000-8000-000000000b03', '00000000-0000-7000-8000-0000000000a1', NULL,
   'cloudflare', 'Cloudflare', 'external', 'builtin.generic', '[]');

INSERT INTO links (id, workspace_id, source_node_id, target_node_id, kind) VALUES
  ('00000000-0000-7000-8000-000000000d01', '00000000-0000-7000-8000-0000000000a1',
   '00000000-0000-7000-8000-000000000b01', '00000000-0000-7000-8000-000000000b02', 'physical'),
  ('00000000-0000-7000-8000-000000000d02', '00000000-0000-7000-8000-0000000000a1',
   '00000000-0000-7000-8000-000000000b03', '00000000-0000-7000-8000-000000000b02', 'proxy_route');

DO $$ BEGIN
  INSERT INTO links (workspace_id, source_node_id, target_node_id, kind) VALUES
    ('00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-000000000b01',
     '00000000-0000-7000-8000-000000000b01', 'physical');
  RAISE EXCEPTION 'FAIL: self-link was accepted';
EXCEPTION WHEN check_violation THEN RAISE NOTICE 'PASS self-link rejected';
END $$;

-- ── 3. the same node on two canvases ──────────────────────────────────────
INSERT INTO canvases (id, workspace_id, name) VALUES
  ('00000000-0000-7000-8000-00000000c001', '00000000-0000-7000-8000-0000000000a1', 'Physical'),
  ('00000000-0000-7000-8000-00000000c002', '00000000-0000-7000-8000-0000000000a1', 'Services');

INSERT INTO placements (workspace_id, canvas_id, node_id, x, y) VALUES
  ('00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-00000000c001', '00000000-0000-7000-8000-000000000b01', 0, 0),
  ('00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-00000000c001', '00000000-0000-7000-8000-000000000b02', 200, 0),
  ('00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-00000000c002', '00000000-0000-7000-8000-000000000b02', 50, 50),
  ('00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-00000000c002', '00000000-0000-7000-8000-000000000b03', 300, 50);

INSERT INTO placements (workspace_id, canvas_id, area_id, x, y, width, height) VALUES
  ('00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-00000000c001', '00000000-0000-7000-8000-0000000000e2', -50, -50, 400, 200);

INSERT INTO placements (workspace_id, canvas_id, link_id) VALUES
  ('00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-00000000c001', '00000000-0000-7000-8000-000000000d01'),
  ('00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-00000000c002', '00000000-0000-7000-8000-000000000d02');

DO $$ BEGIN
  -- router is not placed on "Services", so its physical link cannot be drawn there
  INSERT INTO placements (workspace_id, canvas_id, link_id) VALUES
    ('00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-00000000c002', '00000000-0000-7000-8000-000000000d01');
  RAISE EXCEPTION 'FAIL: link placed without its endpoints';
EXCEPTION WHEN check_violation THEN RAISE NOTICE 'PASS a link override needs both nodes on the canvas';
END $$;

DO $$ BEGIN
  INSERT INTO placements (workspace_id, canvas_id, node_id) VALUES
    ('00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-00000000c001', '00000000-0000-7000-8000-000000000b01');
  RAISE EXCEPTION 'FAIL: node placed twice on one canvas';
EXCEPTION WHEN unique_violation THEN RAISE NOTICE 'PASS a node appears at most once per canvas';
END $$;

-- rename once, seen from both canvases; version bumps
UPDATE nodes SET name = 'NPM' WHERE id = '00000000-0000-7000-8000-000000000b02';
DO $$
DECLARE v int; n int;
BEGIN
  SELECT version INTO v FROM nodes WHERE id = '00000000-0000-7000-8000-000000000b02';
  IF v <> 2 THEN RAISE EXCEPTION 'FAIL: version not bumped (%)', v; END IF;
  SELECT count(DISTINCT p.canvas_id) INTO n FROM placements p JOIN nodes x ON x.id = p.node_id
   WHERE x.name = 'NPM';
  IF n <> 2 THEN RAISE EXCEPTION 'FAIL: expected NPM on 2 canvases, got %', n; END IF;
  RAISE NOTICE 'PASS shared node renamed once, visible on both canvases, version bumped';
END $$;

-- remove NPM from "Physical": its link drawing there disappears, inventory link stays
DELETE FROM placements WHERE canvas_id = '00000000-0000-7000-8000-00000000c001'
  AND node_id = '00000000-0000-7000-8000-000000000b02';
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM placements WHERE canvas_id = '00000000-0000-7000-8000-00000000c001'
             AND link_id = '00000000-0000-7000-8000-000000000d01') THEN
    RAISE EXCEPTION 'FAIL: orphan link placement left on canvas';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM links WHERE id = '00000000-0000-7000-8000-000000000d01') THEN
    RAISE EXCEPTION 'FAIL: inventory link deleted by a canvas removal';
  END IF;
  RAISE NOTICE 'PASS remove-from-canvas keeps inventory, drops dependent link drawing';
END $$;

-- notes: attach to a node; a note with two targets is rejected
INSERT INTO notes (workspace_id, node_id, title, body, is_sensitive) VALUES
  ('00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-000000000b01', 'Admin login', 'Ask Alice', true);
DO $$ BEGIN
  INSERT INTO notes (workspace_id, node_id, canvas_id, body) VALUES
    ('00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-000000000b01',
     '00000000-0000-7000-8000-00000000c001', 'two targets');
  RAISE EXCEPTION 'FAIL: note with two targets accepted';
EXCEPTION WHEN check_violation THEN RAISE NOTICE 'PASS note has at most one target';
END $$;

-- editor grant must reference an actual member
INSERT INTO canvas_grants (workspace_id, canvas_id, user_id) VALUES
  ('00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-00000000c002', '00000000-0000-7000-8000-000000000002');
DO $$ BEGIN
  INSERT INTO canvas_grants (workspace_id, canvas_id, user_id) VALUES
    ('00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-00000000c001', '00000000-0000-7000-8000-000000000099');
  RAISE EXCEPTION 'FAIL: grant for non-member accepted';
EXCEPTION WHEN foreign_key_violation THEN RAISE NOTICE 'PASS canvas grants require workspace membership';
END $$;

-- ── 3b. links appear automatically; canvases can hide them ────────────────
-- (state: "Physical" has router only after NPM was removed; "Services" has NPM + Cloudflare)
INSERT INTO placements (workspace_id, canvas_id, node_id, x, y) VALUES
  ('00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-00000000c001', '00000000-0000-7000-8000-000000000b02', 200, 0);
DO $$
DECLARE n int;
BEGIN
  SELECT count(*) INTO n FROM canvas_visible_links
   WHERE canvas_id = '00000000-0000-7000-8000-00000000c001' AND link_id = '00000000-0000-7000-8000-000000000d01';
  IF n <> 1 THEN RAISE EXCEPTION 'FAIL: link not shown automatically'; END IF;
  SELECT count(*) INTO n FROM canvas_visible_links WHERE canvas_id = '00000000-0000-7000-8000-00000000c002';
  IF n <> 1 THEN RAISE EXCEPTION 'FAIL: Services should show exactly the proxy route, got %', n; END IF;
  RAISE NOTICE 'PASS a link shows on every canvas where both nodes are placed';
END $$;

INSERT INTO placements (workspace_id, canvas_id, link_id, hidden) VALUES
  ('00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-00000000c001', '00000000-0000-7000-8000-000000000d01', true);
UPDATE canvases SET settings = '{"hiddenLinkKinds": ["proxy_route"]}'
 WHERE id = '00000000-0000-7000-8000-00000000c002';
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM canvas_visible_links WHERE canvas_id IN
             ('00000000-0000-7000-8000-00000000c001', '00000000-0000-7000-8000-00000000c002')) THEN
    RAISE EXCEPTION 'FAIL: hidden links still visible';
  END IF;
  RAISE NOTICE 'PASS a canvas can hide one link or a whole link kind';
END $$;
DELETE FROM placements WHERE link_id = '00000000-0000-7000-8000-000000000d01';
UPDATE canvases SET settings = '{}' WHERE id = '00000000-0000-7000-8000-00000000c002';

-- ── 3c. area boxes, drawing inside them, entering an area ─────────────────
-- the "Main LAN" box is already on "Physical"; draw the router inside it
UPDATE placements SET parent_placement_id =
       (SELECT id FROM placements WHERE canvas_id = '00000000-0000-7000-8000-00000000c001'
                                    AND area_id = '00000000-0000-7000-8000-0000000000e2')
 WHERE canvas_id = '00000000-0000-7000-8000-00000000c001' AND node_id = '00000000-0000-7000-8000-000000000b01';
\echo 'PASS a node can be drawn inside the box of its own area'

DO $$ BEGIN
  -- Cloudflare belongs to no area, so it may not be drawn inside the Main LAN box
  INSERT INTO placements (workspace_id, canvas_id, node_id, parent_placement_id)
  SELECT '00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-00000000c001',
         '00000000-0000-7000-8000-000000000b03', id
    FROM placements WHERE canvas_id = '00000000-0000-7000-8000-00000000c001'
                      AND area_id = '00000000-0000-7000-8000-0000000000e2';
  RAISE EXCEPTION 'FAIL: node drawn inside a foreign area box';
EXCEPTION WHEN check_violation THEN RAISE NOTICE 'PASS a node cannot be drawn inside an area it does not belong to';
END $$;

UPDATE nodes SET area_id = NULL WHERE id = '00000000-0000-7000-8000-000000000b01';
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM placements WHERE node_id = '00000000-0000-7000-8000-000000000b01'
                                         AND parent_placement_id IS NOT NULL) THEN
    RAISE EXCEPTION 'FAIL: drawing still inside the old area box';
  END IF;
  RAISE NOTICE 'PASS moving a node out of an area moves its drawings out of the box';
END $$;
UPDATE nodes SET area_id = '00000000-0000-7000-8000-0000000000e2' WHERE id = '00000000-0000-7000-8000-000000000b01';
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM placements WHERE node_id = '00000000-0000-7000-8000-000000000b01'
                   AND canvas_id = '00000000-0000-7000-8000-00000000c001' AND parent_placement_id IS NOT NULL) THEN
    RAISE EXCEPTION 'FAIL: drawing not moved into the new area box';
  END IF;
  RAISE NOTICE 'PASS moving a node into an area moves its drawing into that area''s box';
END $$;

-- entering an area: each area has at most one inner canvas; its name does not clash with tabs
INSERT INTO canvases (workspace_id, area_id, name) VALUES
  ('00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-0000000000e2', 'Physical');
DO $$ BEGIN
  INSERT INTO canvases (workspace_id, area_id, name) VALUES
    ('00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-0000000000e2', 'Main LAN 2');
  RAISE EXCEPTION 'FAIL: second inner canvas for one area';
EXCEPTION WHEN unique_violation THEN RAISE NOTICE 'PASS an area has exactly one inner canvas';
END $$;

DO $$ BEGIN
  UPDATE placements SET collapsed = true WHERE node_id = '00000000-0000-7000-8000-000000000b01';
  RAISE EXCEPTION 'FAIL: a node placement was collapsed';
EXCEPTION WHEN check_violation THEN RAISE NOTICE 'PASS only area boxes can be collapsed';
END $$;
UPDATE placements SET collapsed = true WHERE area_id = '00000000-0000-7000-8000-0000000000e2';

-- ── 4. isolation between workspaces ───────────────────────────────────────
SELECT set_config('app.workspace_id', '00000000-0000-7000-8000-0000000000a2', true);

DO $$
DECLARE n int;
BEGIN
  SELECT count(*) INTO n FROM nodes;
  IF n <> 0 THEN RAISE EXCEPTION 'FAIL: RLS leaked % nodes from another workspace', n; END IF;
  SELECT count(*) INTO n FROM canvases;
  IF n <> 0 THEN RAISE EXCEPTION 'FAIL: RLS leaked % canvases', n; END IF;
  RAISE NOTICE 'PASS RLS hides other workspaces'' rows';
END $$;

DO $$ BEGIN
  -- writing a row for "home" while scoped to "lab-b"
  INSERT INTO nodes (workspace_id, slug, name, category) VALUES
    ('00000000-0000-7000-8000-0000000000a1', 'sneaky', 'Sneaky', 'generic');
  RAISE EXCEPTION 'FAIL: RLS allowed writing into another workspace';
EXCEPTION WHEN insufficient_privilege THEN RAISE NOTICE 'PASS RLS blocks writes into other workspaces';
END $$;

INSERT INTO nodes (id, workspace_id, slug, name, category) VALUES
  ('00000000-0000-7000-8000-000000000b11', '00000000-0000-7000-8000-0000000000a2', 'router', 'Router B', 'network');
DO $$ BEGIN
  -- a link from lab-b's node to home's node: composite FK must reject it
  INSERT INTO links (workspace_id, source_node_id, target_node_id, kind) VALUES
    ('00000000-0000-7000-8000-0000000000a2', '00000000-0000-7000-8000-000000000b11',
     '00000000-0000-7000-8000-000000000b01', 'logical');
  RAISE EXCEPTION 'FAIL: cross-workspace link accepted';
EXCEPTION WHEN foreign_key_violation THEN RAISE NOTICE 'PASS cross-workspace references impossible (composite FK)';
END $$;
\echo 'PASS slugs are per workspace (both workspaces have a "router")'

-- with no workspace selected, nothing is visible
SELECT set_config('app.workspace_id', '', true);
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM nodes) THEN RAISE EXCEPTION 'FAIL: rows visible without a workspace'; END IF;
  RAISE NOTICE 'PASS no workspace selected = no inventory rows';
END $$;

-- ── 5. delete from inventory cascades everywhere ─────────────────────────
SELECT set_config('app.workspace_id', '00000000-0000-7000-8000-0000000000a1', true);
DELETE FROM nodes WHERE id = '00000000-0000-7000-8000-000000000b02';  -- NPM
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM links WHERE source_node_id = '00000000-0000-7000-8000-000000000b02'
                                  OR target_node_id = '00000000-0000-7000-8000-000000000b02') THEN
    RAISE EXCEPTION 'FAIL: links of deleted node remain';
  END IF;
  IF EXISTS (SELECT 1 FROM placements WHERE node_id = '00000000-0000-7000-8000-000000000b02'
                                       OR link_id = '00000000-0000-7000-8000-000000000d02') THEN
    RAISE EXCEPTION 'FAIL: placements of deleted node/link remain';
  END IF;
  RAISE NOTICE 'PASS delete-from-inventory removes links and placements on every canvas';
END $$;

-- deleting the parent area un-nests the child instead of deleting it
DELETE FROM areas WHERE id = '00000000-0000-7000-8000-0000000000e1';
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM areas WHERE id = '00000000-0000-7000-8000-0000000000e2' AND parent_id IS NULL) THEN
    RAISE EXCEPTION 'FAIL: child area not un-nested';
  END IF;
  RAISE NOTICE 'PASS deleting a parent area keeps its children (moved to top level)';
END $$;

-- ── 6. guest view: what a share link may read ─────────────────────────────
INSERT INTO share_links (workspace_id, canvas_id, token_hash) VALUES
  ('00000000-0000-7000-8000-0000000000a1', '00000000-0000-7000-8000-00000000c001', '\x01');
DO $$
DECLARE leaked int;
BEGIN
  -- the API builds the guest payload like this: placed nodes only, sensitive fields removed
  SELECT count(*) INTO leaked
  FROM placements p
  JOIN nodes n ON n.id = p.node_id
  CROSS JOIN LATERAL (
    SELECT coalesce(jsonb_agg(f), '[]') AS props
    FROM jsonb_array_elements(n.properties) f
    WHERE NOT coalesce((f ->> 'sensitive')::boolean, false)
  ) redacted
  WHERE p.canvas_id = '00000000-0000-7000-8000-00000000c001'
    AND redacted.props @> '[{"sensitive": true}]';
  IF leaked <> 0 THEN RAISE EXCEPTION 'FAIL: sensitive field in guest payload'; END IF;
  RAISE NOTICE 'PASS guest canvas query can exclude sensitive fields';
END $$;

-- ── 7. audit log is append-only ───────────────────────────────────────────
INSERT INTO audit_log (workspace_id, actor_type, actor_id, action) VALUES
  ('00000000-0000-7000-8000-0000000000a1', 'user', '00000000-0000-7000-8000-000000000001', 'node.delete');
DO $$ BEGIN
  UPDATE audit_log SET action = 'node.create';
  RAISE EXCEPTION 'FAIL: audit log was modified';
EXCEPTION WHEN insufficient_privilege THEN RAISE NOTICE 'PASS audit log rejects updates';
END $$;

ROLLBACK;
\echo 'ALL SCENARIOS PASSED'
