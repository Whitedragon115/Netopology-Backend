-- Netopology schema v1 (phase A1).
-- Requires PostgreSQL 18+ (uuidv7()). Design view: docs/design/schema.dbml.

-- +goose Up

-- ───────────── helpers ─────────────

-- +goose StatementBegin
CREATE FUNCTION touch_updated_at() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  NEW.updated_at := now();
  RETURN NEW;
END $$;
-- +goose StatementEnd

-- For tables with optimistic concurrency: bump version on every update.
-- +goose StatementBegin
CREATE FUNCTION touch_versioned() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  NEW.updated_at := now();
  NEW.version := OLD.version + 1;
  RETURN NEW;
END $$;
-- +goose StatementEnd

-- ───────────── accounts ─────────────

CREATE TABLE users (
  id                uuid PRIMARY KEY DEFAULT uuidv7(),
  username          text NOT NULL CHECK (username ~ '^[A-Za-z0-9][A-Za-z0-9._-]{1,38}[A-Za-z0-9]$'),
  email             text NULL CHECK (email IS NULL OR email ~ '^[^@\s]+@[^@\s]+$'),
  display_name      text NOT NULL DEFAULT '' CHECK (length(display_name) <= 100),
  password_hash     text NULL,  -- NULL = this account signs in only through Google/Discord
  instance_role     text NOT NULL DEFAULT 'member' CHECK (instance_role IN ('owner', 'admin', 'member')),
  status            text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
  last_login_at     timestamptz NULL,
  created_at        timestamptz NOT NULL DEFAULT now(),
  updated_at        timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_username_key ON users (lower(username));
CREATE UNIQUE INDEX users_email_key ON users (lower(email)) WHERE email IS NOT NULL;
-- exactly one server owner may exist (the account created at first-run setup)
CREATE UNIQUE INDEX users_single_instance_owner ON users ((true)) WHERE instance_role = 'owner';
CREATE TRIGGER users_touch BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- External sign-in identities linked to an account. No self-signup: an identity can only be
-- linked to an account that already exists (created by a server owner/admin).
CREATE TABLE user_identities (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  provider     text NOT NULL CHECK (provider IN ('google', 'discord')),
  subject      text NOT NULL,  -- the provider's stable user id (Google "sub", Discord user id)
  email        text NULL,      -- as reported (and verified) by the provider, for display
  display_name text NULL,
  created_at   timestamptz NOT NULL DEFAULT now(),
  last_used_at timestamptz NULL,
  UNIQUE (provider, subject),
  UNIQUE (user_id, provider)
);

CREATE TABLE sessions (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  token_hash   bytea NOT NULL UNIQUE,
  ip           inet NULL,
  user_agent   text NULL,
  created_at   timestamptz NOT NULL DEFAULT now(),
  last_seen_at timestamptz NOT NULL DEFAULT now(),
  expires_at   timestamptz NOT NULL
);
CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE instance_settings (
  key        text PRIMARY KEY,
  value      jsonb NOT NULL,
  updated_by uuid NULL REFERENCES users (id) ON DELETE SET NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER instance_settings_touch BEFORE UPDATE ON instance_settings FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- ───────────── workspaces & membership ─────────────

CREATE TABLE workspaces (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  name         text NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
  slug         text NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9][a-z0-9-]{0,62}$'),
  description  text NOT NULL DEFAULT '',
  is_protected boolean NOT NULL DEFAULT false,
  settings     jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(settings) = 'object'),
  created_by   uuid NULL REFERENCES users (id) ON DELETE SET NULL,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  version      integer NOT NULL DEFAULT 1
);
CREATE TRIGGER workspaces_touch BEFORE UPDATE ON workspaces FOR EACH ROW EXECUTE FUNCTION touch_versioned();

CREATE TABLE workspace_members (
  workspace_id       uuid NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  user_id            uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  role               text NOT NULL CHECK (role IN ('owner', 'admin', 'editor', 'viewer')),
  can_view_sensitive boolean NOT NULL DEFAULT false,
  added_by           uuid NULL REFERENCES users (id) ON DELETE SET NULL,
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (workspace_id, user_id)
);
CREATE INDEX workspace_members_user_id_idx ON workspace_members (user_id);
CREATE TRIGGER workspace_members_touch BEFORE UPDATE ON workspace_members FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- ───────────── inventory ─────────────

CREATE TABLE areas (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  workspace_id uuid NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  parent_id    uuid NULL,
  name         text NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
  kind         text NOT NULL DEFAULT 'custom' CHECK (kind IN ('site', 'network', 'vlan', 'cloud', 'custom')),
  color        text NOT NULL DEFAULT 'yellow' CHECK (color IN ('yellow', 'teal', 'violet', 'rose', 'blue', 'green', 'gray')),
  cidr         cidr NULL,
  vlan_id      integer NULL CHECK (vlan_id BETWEEN 1 AND 4094),
  description  text NOT NULL DEFAULT '',
  properties   jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(properties) = 'array'),
  created_by   uuid NULL REFERENCES users (id) ON DELETE SET NULL,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  version      integer NOT NULL DEFAULT 1,
  UNIQUE (workspace_id, id),
  CHECK (parent_id IS DISTINCT FROM id),
  FOREIGN KEY (workspace_id, parent_id) REFERENCES areas (workspace_id, id) ON DELETE SET NULL (parent_id)
);
CREATE INDEX areas_parent_idx ON areas (workspace_id, parent_id);
CREATE TRIGGER areas_touch BEFORE UPDATE ON areas FOR EACH ROW EXECUTE FUNCTION touch_versioned();

-- Reject area nesting cycles (A inside B inside A).
-- +goose StatementBegin
CREATE FUNCTION areas_prevent_cycle() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.parent_id IS NULL THEN
    RETURN NEW;
  END IF;
  IF EXISTS (
    WITH RECURSIVE ancestors (id, parent_id) AS (
      SELECT a.id, a.parent_id FROM areas a WHERE a.id = NEW.parent_id
      UNION
      SELECT a.id, a.parent_id FROM areas a JOIN ancestors s ON a.id = s.parent_id
    )
    SELECT 1 FROM ancestors WHERE id = NEW.id
  ) THEN
    RAISE EXCEPTION 'area % cannot be nested inside its own descendant', NEW.id
      USING ERRCODE = 'check_violation', CONSTRAINT = 'areas_no_cycle';
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER areas_no_cycle BEFORE INSERT OR UPDATE OF parent_id ON areas
  FOR EACH ROW EXECUTE FUNCTION areas_prevent_cycle();

CREATE TABLE nodes (
  id                uuid PRIMARY KEY DEFAULT uuidv7(),
  workspace_id      uuid NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  area_id           uuid NULL,
  slug              text NOT NULL CHECK (slug ~ '^[a-z0-9][a-z0-9-]{0,62}$'),
  name              text NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
  category          text NOT NULL CHECK (category IN ('network', 'compute', 'service', 'storage', 'external', 'generic')),
  type              text NOT NULL DEFAULT 'builtin.generic',
  icon              text NULL,
  description       text NOT NULL DEFAULT '',
  properties        jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(properties) = 'array'),
  health_status     text NOT NULL DEFAULT 'unknown' CHECK (health_status IN ('healthy', 'degraded', 'down', 'unknown')),
  health_source     text NOT NULL DEFAULT 'manual' CHECK (health_source IN ('manual', 'plugin')),
  health_message    text NULL,
  health_checked_at timestamptz NULL,
  is_protected      boolean NOT NULL DEFAULT false,
  created_by        uuid NULL REFERENCES users (id) ON DELETE SET NULL,
  created_at        timestamptz NOT NULL DEFAULT now(),
  updated_at        timestamptz NOT NULL DEFAULT now(),
  version           integer NOT NULL DEFAULT 1,
  UNIQUE (workspace_id, id),
  UNIQUE (workspace_id, slug),
  FOREIGN KEY (workspace_id, area_id) REFERENCES areas (workspace_id, id) ON DELETE SET NULL (area_id)
);
CREATE INDEX nodes_area_idx ON nodes (workspace_id, area_id);
CREATE INDEX nodes_category_idx ON nodes (workspace_id, category);
CREATE TRIGGER nodes_touch BEFORE UPDATE ON nodes FOR EACH ROW EXECUTE FUNCTION touch_versioned();

CREATE TABLE links (
  id                uuid PRIMARY KEY DEFAULT uuidv7(),
  workspace_id      uuid NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  source_node_id    uuid NOT NULL,
  target_node_id    uuid NOT NULL,
  kind              text NOT NULL CHECK (kind IN ('physical', 'vpn_tunnel', 'proxy_route', 'dns_dependency', 'storage_mount', 'logical')),
  directed          boolean NOT NULL DEFAULT false,
  label             text NOT NULL DEFAULT '',
  source_port       text NULL,
  target_port       text NULL,
  properties        jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(properties) = 'array'),
  health_status     text NOT NULL DEFAULT 'unknown' CHECK (health_status IN ('healthy', 'degraded', 'down', 'unknown')),
  health_source     text NOT NULL DEFAULT 'manual' CHECK (health_source IN ('manual', 'plugin')),
  health_message    text NULL,
  health_checked_at timestamptz NULL,
  created_by        uuid NULL REFERENCES users (id) ON DELETE SET NULL,
  created_at        timestamptz NOT NULL DEFAULT now(),
  updated_at        timestamptz NOT NULL DEFAULT now(),
  version           integer NOT NULL DEFAULT 1,
  UNIQUE (workspace_id, id),
  CHECK (source_node_id <> target_node_id),
  FOREIGN KEY (workspace_id, source_node_id) REFERENCES nodes (workspace_id, id) ON DELETE CASCADE,
  FOREIGN KEY (workspace_id, target_node_id) REFERENCES nodes (workspace_id, id) ON DELETE CASCADE
);
CREATE INDEX links_source_idx ON links (workspace_id, source_node_id);
CREATE INDEX links_target_idx ON links (workspace_id, target_node_id);
CREATE TRIGGER links_touch BEFORE UPDATE ON links FOR EACH ROW EXECUTE FUNCTION touch_versioned();

-- ───────────── canvases ─────────────

CREATE TABLE canvases (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  workspace_id uuid NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  area_id      uuid NULL,  -- set = this is the inner canvas of an area ("enter the area"), not a tab
  name         text NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
  description  text NOT NULL DEFAULT '',
  sort_order   integer NOT NULL DEFAULT 0,
  viewport     jsonb NOT NULL DEFAULT '{"x":0,"y":0,"zoom":1}' CHECK (jsonb_typeof(viewport) = 'object'),
  settings     jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(settings) = 'object'),
  is_protected boolean NOT NULL DEFAULT false,
  locked_by    uuid NULL REFERENCES users (id) ON DELETE SET NULL,
  locked_at    timestamptz NULL,
  created_by   uuid NULL REFERENCES users (id) ON DELETE SET NULL,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  version      integer NOT NULL DEFAULT 1,
  UNIQUE (workspace_id, id),
  CHECK ((locked_by IS NULL) = (locked_at IS NULL)),
  FOREIGN KEY (workspace_id, area_id) REFERENCES areas (workspace_id, id) ON DELETE CASCADE
);
-- tab canvases have unique names; area canvases are named after their area
CREATE UNIQUE INDEX canvases_name_key ON canvases (workspace_id, name) WHERE area_id IS NULL;
CREATE UNIQUE INDEX canvases_area_key ON canvases (area_id) WHERE area_id IS NOT NULL;
CREATE TRIGGER canvases_touch BEFORE UPDATE ON canvases FOR EACH ROW EXECUTE FUNCTION touch_versioned();

CREATE TABLE canvas_grants (
  workspace_id uuid NOT NULL,
  canvas_id    uuid NOT NULL,
  user_id      uuid NOT NULL,
  granted_by   uuid NULL REFERENCES users (id) ON DELETE SET NULL,
  created_at   timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (canvas_id, user_id),
  FOREIGN KEY (workspace_id, canvas_id) REFERENCES canvases (workspace_id, id) ON DELETE CASCADE,
  FOREIGN KEY (workspace_id, user_id) REFERENCES workspace_members (workspace_id, user_id) ON DELETE CASCADE
);
CREATE INDEX canvas_grants_user_idx ON canvas_grants (workspace_id, user_id);

-- notes come after canvases because a note can target a canvas
CREATE TABLE notes (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  workspace_id uuid NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  node_id      uuid NULL,
  link_id      uuid NULL,
  area_id      uuid NULL,
  canvas_id    uuid NULL,
  kind         text NOT NULL DEFAULT 'note' CHECK (kind IN ('note', 'rule')),
  title        text NOT NULL DEFAULT '' CHECK (length(title) <= 200),
  body         text NOT NULL DEFAULT '',
  is_sensitive boolean NOT NULL DEFAULT false,
  color        text NOT NULL DEFAULT 'default' CHECK (color IN ('default', 'yellow', 'teal', 'violet', 'rose')),
  created_by   uuid NULL REFERENCES users (id) ON DELETE SET NULL,
  updated_by   uuid NULL REFERENCES users (id) ON DELETE SET NULL,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  version      integer NOT NULL DEFAULT 1,
  UNIQUE (workspace_id, id),
  CHECK (num_nonnulls(node_id, link_id, area_id, canvas_id) <= 1),
  FOREIGN KEY (workspace_id, node_id) REFERENCES nodes (workspace_id, id) ON DELETE CASCADE,
  FOREIGN KEY (workspace_id, link_id) REFERENCES links (workspace_id, id) ON DELETE CASCADE,
  FOREIGN KEY (workspace_id, area_id) REFERENCES areas (workspace_id, id) ON DELETE CASCADE,
  FOREIGN KEY (workspace_id, canvas_id) REFERENCES canvases (workspace_id, id) ON DELETE CASCADE
);
CREATE INDEX notes_node_idx ON notes (workspace_id, node_id) WHERE node_id IS NOT NULL;
CREATE INDEX notes_link_idx ON notes (workspace_id, link_id) WHERE link_id IS NOT NULL;
CREATE INDEX notes_area_idx ON notes (workspace_id, area_id) WHERE area_id IS NOT NULL;
CREATE INDEX notes_canvas_idx ON notes (workspace_id, canvas_id) WHERE canvas_id IS NOT NULL;
CREATE TRIGGER notes_touch BEFORE UPDATE ON notes FOR EACH ROW EXECUTE FUNCTION touch_versioned();

CREATE TABLE placements (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  workspace_id uuid NOT NULL,
  canvas_id    uuid NOT NULL,
  node_id      uuid NULL,
  area_id      uuid NULL,
  note_id      uuid NULL,
  link_id      uuid NULL,  -- link rows are OVERRIDES only (hide / restyle / route); links show automatically
  parent_placement_id uuid NULL,  -- drawn inside this area box (coordinates relative to it)
  x            double precision NOT NULL DEFAULT 0,
  y            double precision NOT NULL DEFAULT 0,
  width        double precision NULL CHECK (width IS NULL OR width > 0),
  height       double precision NULL CHECK (height IS NULL OR height > 0),
  z_index      integer NOT NULL DEFAULT 0,
  collapsed    boolean NOT NULL DEFAULT false,  -- area boxes: shown as one block you can enter
  hidden       boolean NOT NULL DEFAULT false,  -- link overrides: hide this link on this canvas
  style        jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(style) = 'object'),
  waypoints    jsonb NULL CHECK (waypoints IS NULL OR jsonb_typeof(waypoints) = 'array'),
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  UNIQUE (workspace_id, id),
  UNIQUE (workspace_id, canvas_id, id),
  CHECK (num_nonnulls(node_id, area_id, note_id, link_id) = 1),
  CHECK (waypoints IS NULL OR link_id IS NOT NULL),
  CHECK (NOT collapsed OR area_id IS NOT NULL),
  CHECK (NOT hidden OR link_id IS NOT NULL),
  CHECK (parent_placement_id IS NULL OR link_id IS NULL),
  CHECK (parent_placement_id IS DISTINCT FROM id),
  FOREIGN KEY (workspace_id, canvas_id) REFERENCES canvases (workspace_id, id) ON DELETE CASCADE,
  FOREIGN KEY (workspace_id, canvas_id, parent_placement_id)
    REFERENCES placements (workspace_id, canvas_id, id) ON DELETE SET NULL (parent_placement_id),
  FOREIGN KEY (workspace_id, node_id) REFERENCES nodes (workspace_id, id) ON DELETE CASCADE,
  FOREIGN KEY (workspace_id, area_id) REFERENCES areas (workspace_id, id) ON DELETE CASCADE,
  FOREIGN KEY (workspace_id, note_id) REFERENCES notes (workspace_id, id) ON DELETE CASCADE,
  FOREIGN KEY (workspace_id, link_id) REFERENCES links (workspace_id, id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX placements_canvas_node_key ON placements (canvas_id, node_id) WHERE node_id IS NOT NULL;
CREATE UNIQUE INDEX placements_canvas_area_key ON placements (canvas_id, area_id) WHERE area_id IS NOT NULL;
CREATE UNIQUE INDEX placements_canvas_note_key ON placements (canvas_id, note_id) WHERE note_id IS NOT NULL;
CREATE UNIQUE INDEX placements_canvas_link_key ON placements (canvas_id, link_id) WHERE link_id IS NOT NULL;
CREATE INDEX placements_canvas_idx ON placements (workspace_id, canvas_id);
CREATE INDEX placements_node_idx ON placements (workspace_id, node_id) WHERE node_id IS NOT NULL;
CREATE INDEX placements_link_idx ON placements (workspace_id, link_id) WHERE link_id IS NOT NULL;
CREATE TRIGGER placements_touch BEFORE UPDATE ON placements FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- A link can only be drawn on a canvas where both of its nodes are placed.
-- +goose StatementBegin
CREATE FUNCTION placements_check_link_endpoints() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
  missing integer;
BEGIN
  IF NEW.link_id IS NULL THEN
    RETURN NEW;
  END IF;
  SELECT count(*) INTO missing
  FROM links l
  CROSS JOIN LATERAL (VALUES (l.source_node_id), (l.target_node_id)) AS ends (node_id)
  WHERE l.workspace_id = NEW.workspace_id AND l.id = NEW.link_id
    AND NOT EXISTS (
      SELECT 1 FROM placements p
      WHERE p.canvas_id = NEW.canvas_id AND p.node_id = ends.node_id
    );
  IF missing > 0 THEN
    RAISE EXCEPTION 'link % needs both endpoint nodes placed on canvas %', NEW.link_id, NEW.canvas_id
      USING ERRCODE = 'check_violation', CONSTRAINT = 'placements_link_endpoints';
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER placements_link_endpoints BEFORE INSERT OR UPDATE OF link_id, canvas_id ON placements
  FOR EACH ROW EXECUTE FUNCTION placements_check_link_endpoints();

-- Removing a node from a canvas also removes that canvas's drawings of links touching it.
-- +goose StatementBegin
CREATE FUNCTION placements_cascade_link_removal() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  IF OLD.node_id IS NULL THEN
    RETURN OLD;
  END IF;
  DELETE FROM placements p
  USING links l
  WHERE p.canvas_id = OLD.canvas_id
    AND p.link_id = l.id
    AND l.workspace_id = OLD.workspace_id
    AND (l.source_node_id = OLD.node_id OR l.target_node_id = OLD.node_id);
  RETURN OLD;
END $$;
-- +goose StatementEnd
CREATE TRIGGER placements_link_removal AFTER DELETE ON placements
  FOR EACH ROW EXECUTE FUNCTION placements_cascade_link_removal();

-- Drawing must match the inventory: an item can only be drawn inside the box of the area it
-- belongs to (node.area_id, or area.parent_id for nested area boxes). Notes may sit in any box.
-- +goose StatementBegin
CREATE FUNCTION placements_check_parent() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
  box_area uuid;
  own_area uuid;
BEGIN
  IF NEW.parent_placement_id IS NULL THEN
    RETURN NEW;
  END IF;
  SELECT p.area_id INTO box_area FROM placements p WHERE p.id = NEW.parent_placement_id;
  IF box_area IS NULL THEN
    RAISE EXCEPTION 'placement % can only be drawn inside an area box', NEW.id
      USING ERRCODE = 'check_violation', CONSTRAINT = 'placements_parent_is_area';
  END IF;
  IF NEW.node_id IS NOT NULL THEN
    SELECT n.area_id INTO own_area FROM nodes n WHERE n.id = NEW.node_id;
  ELSIF NEW.area_id IS NOT NULL THEN
    SELECT a.parent_id INTO own_area FROM areas a WHERE a.id = NEW.area_id;
  ELSE
    RETURN NEW;
  END IF;
  IF own_area IS DISTINCT FROM box_area THEN
    RAISE EXCEPTION 'item is drawn inside area % but belongs to area %', box_area, own_area
      USING ERRCODE = 'check_violation', CONSTRAINT = 'placements_parent_matches_area';
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER placements_parent BEFORE INSERT OR UPDATE OF parent_placement_id, node_id, area_id ON placements
  FOR EACH ROW EXECUTE FUNCTION placements_check_parent();

-- When a node moves to another area, re-parent its drawings: inside the new area's box on
-- canvases where that box exists, otherwise free. (The API also converts the coordinates.)
-- +goose StatementBegin
CREATE FUNCTION nodes_reparent_placements() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  UPDATE placements p
     SET parent_placement_id = (SELECT b.id FROM placements b
                                 WHERE b.canvas_id = p.canvas_id AND b.area_id = NEW.area_id)
   WHERE p.workspace_id = NEW.workspace_id AND p.node_id = NEW.id;
  RETURN NULL;
END $$;
-- +goose StatementEnd
CREATE TRIGGER nodes_reparent AFTER UPDATE OF area_id ON nodes
  FOR EACH ROW WHEN (OLD.area_id IS DISTINCT FROM NEW.area_id)
  EXECUTE FUNCTION nodes_reparent_placements();

-- Same for an area moved under another parent area.
-- +goose StatementBegin
CREATE FUNCTION areas_reparent_placements() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  UPDATE placements p
     SET parent_placement_id = (SELECT b.id FROM placements b
                                 WHERE b.canvas_id = p.canvas_id AND b.area_id = NEW.parent_id)
   WHERE p.workspace_id = NEW.workspace_id AND p.area_id = NEW.id;
  RETURN NULL;
END $$;
-- +goose StatementEnd
CREATE TRIGGER areas_reparent AFTER UPDATE OF parent_id ON areas
  FOR EACH ROW WHEN (OLD.parent_id IS DISTINCT FROM NEW.parent_id)
  EXECUTE FUNCTION areas_reparent_placements();

-- Links appear automatically on every canvas where both of their nodes are placed, unless
-- hidden by an override placement or by the canvas setting {"hiddenLinkKinds": ["physical", ...]}.
-- security_invoker makes the view obey the caller's RLS.
CREATE VIEW canvas_visible_links WITH (security_invoker = true) AS
SELECT c.workspace_id,
       c.id AS canvas_id,
       l.id AS link_id,
       o.id AS override_placement_id
FROM canvases c
JOIN links l ON l.workspace_id = c.workspace_id
JOIN placements ps ON ps.canvas_id = c.id AND ps.node_id = l.source_node_id
JOIN placements pt ON pt.canvas_id = c.id AND pt.node_id = l.target_node_id
LEFT JOIN placements o ON o.canvas_id = c.id AND o.link_id = l.id
WHERE NOT coalesce(o.hidden, false)
  AND NOT (coalesce(c.settings -> 'hiddenLinkKinds', '[]'::jsonb) ? l.kind);

CREATE TABLE canvas_shapes (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  workspace_id uuid NOT NULL,
  canvas_id    uuid NOT NULL,
  kind         text NOT NULL CHECK (kind IN ('text', 'rectangle', 'ellipse', 'line', 'arrow', 'freehand')),
  x            double precision NOT NULL DEFAULT 0,
  y            double precision NOT NULL DEFAULT 0,
  width        double precision NULL,
  height       double precision NULL,
  rotation     double precision NOT NULL DEFAULT 0,
  z_index      integer NOT NULL DEFAULT 0,
  style        jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(style) = 'object'),
  data         jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(data) = 'object'),
  created_by   uuid NULL REFERENCES users (id) ON DELETE SET NULL,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (workspace_id, canvas_id) REFERENCES canvases (workspace_id, id) ON DELETE CASCADE
);
CREATE INDEX canvas_shapes_canvas_idx ON canvas_shapes (workspace_id, canvas_id);
CREATE TRIGGER canvas_shapes_touch BEFORE UPDATE ON canvas_shapes FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- ───────────── sharing & audit ─────────────

CREATE TABLE share_links (
  id            uuid PRIMARY KEY DEFAULT uuidv7(),
  workspace_id  uuid NOT NULL,
  canvas_id     uuid NOT NULL,
  label         text NOT NULL DEFAULT '' CHECK (length(label) <= 100),
  token_hash    bytea NOT NULL UNIQUE,
  password_hash text NULL,
  expires_at    timestamptz NULL,
  revoked_at    timestamptz NULL,
  created_by    uuid NULL REFERENCES users (id) ON DELETE SET NULL,
  created_at    timestamptz NOT NULL DEFAULT now(),
  last_used_at  timestamptz NULL,
  use_count     bigint NOT NULL DEFAULT 0,
  FOREIGN KEY (workspace_id, canvas_id) REFERENCES canvases (workspace_id, id) ON DELETE CASCADE
);
CREATE INDEX share_links_canvas_idx ON share_links (workspace_id, canvas_id);

CREATE TABLE audit_log (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  workspace_id uuid NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  actor_type   text NOT NULL CHECK (actor_type IN ('user', 'share_link', 'system', 'workflow')),
  actor_id     uuid NULL,
  action       text NOT NULL CHECK (action ~ '^[a-z_]+(\.[a-z_]+)+$'),
  target_type  text NULL,
  target_id    uuid NULL,
  metadata     jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(metadata) = 'object'),
  ip           inet NULL,
  user_agent   text NULL,
  created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_log_workspace_time_idx ON audit_log (workspace_id, created_at DESC);
CREATE INDEX audit_log_actor_idx ON audit_log (actor_type, actor_id);

-- +goose StatementBegin
CREATE FUNCTION audit_log_block_update() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'audit_log is append-only' USING ERRCODE = 'insufficient_privilege';
END $$;
-- +goose StatementEnd
CREATE TRIGGER audit_log_append_only BEFORE UPDATE ON audit_log
  FOR EACH ROW EXECUTE FUNCTION audit_log_block_update();

-- +goose Down
DROP VIEW IF EXISTS canvas_visible_links;
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS share_links;
DROP TABLE IF EXISTS canvas_shapes;
DROP TABLE IF EXISTS placements;
DROP TABLE IF EXISTS notes;
DROP TABLE IF EXISTS canvas_grants;
DROP TABLE IF EXISTS canvases;
DROP TABLE IF EXISTS links;
DROP TABLE IF EXISTS nodes;
DROP TABLE IF EXISTS areas;
DROP TABLE IF EXISTS workspace_members;
DROP TABLE IF EXISTS workspaces;
DROP TABLE IF EXISTS instance_settings;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS user_identities;
DROP TABLE IF EXISTS users;
DROP FUNCTION IF EXISTS audit_log_block_update();
DROP FUNCTION IF EXISTS areas_reparent_placements();
DROP FUNCTION IF EXISTS nodes_reparent_placements();
DROP FUNCTION IF EXISTS placements_check_parent();
DROP FUNCTION IF EXISTS placements_cascade_link_removal();
DROP FUNCTION IF EXISTS placements_check_link_endpoints();
DROP FUNCTION IF EXISTS areas_prevent_cycle();
DROP FUNCTION IF EXISTS touch_versioned();
DROP FUNCTION IF EXISTS touch_updated_at();
