-- Live updates (SSE) outbox. See docs/ARCHITECTURE.md §9.
--
-- A service inserts one row per change, in the SAME transaction as the change. The trigger sends a
-- pg_notify on commit (never for a rolled-back change). Each api process LISTENs, and for every
-- subscriber re-reads the changed item AS THAT SUBSCRIBER, so visibility and redaction are exactly
-- the same as a normal GET. Rows are kept 24h so clients can resume with Last-Event-ID.

-- +goose Up
CREATE TABLE workspace_events (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  workspace_id uuid NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
  type         text NOT NULL CHECK (type ~ '^[a-z_]+(\.[a-z_]+)+$'),  -- Event.type in the contract
  subject_id   uuid NULL,                     -- id of the changed item (node, link, canvas, ...)
  canvas_ids   uuid[] NOT NULL DEFAULT '{}',  -- canvases affected, for ?canvasId=; empty = workspace-wide
  actor_id     uuid NULL,                     -- user who caused it (omitted for guests)
  created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX workspace_events_resume_idx ON workspace_events (workspace_id, id);
CREATE INDEX workspace_events_created_idx ON workspace_events (created_at);

-- +goose StatementBegin
CREATE FUNCTION workspace_events_notify() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  -- payload stays tiny (8000 byte limit): "<workspace_id>:<event_id>"
  PERFORM pg_notify('netopology_events', NEW.workspace_id::text || ':' || NEW.id::text);
  RETURN NULL;
END $$;
-- +goose StatementEnd
CREATE TRIGGER workspace_events_notify AFTER INSERT ON workspace_events
  FOR EACH ROW EXECUTE FUNCTION workspace_events_notify();

ALTER TABLE workspace_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE workspace_events FORCE ROW LEVEL SECURITY;
CREATE POLICY workspace_isolation ON workspace_events
  USING (workspace_id = current_workspace_id())
  WITH CHECK (workspace_id = current_workspace_id());

-- Retention runs across all workspaces, which RLS forbids for the app role. This function runs as
-- the schema owner (which has BYPASSRLS, see db/init/10-roles.sh) and can only delete old events.
-- +goose StatementBegin
CREATE FUNCTION prune_workspace_events(keep interval) RETURNS bigint
LANGUAGE sql SECURITY DEFINER SET search_path = public AS $$
  WITH d AS (DELETE FROM workspace_events WHERE created_at < now() - keep RETURNING 1)
  SELECT count(*) FROM d
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION prune_workspace_events(interval) FROM PUBLIC;

-- +goose Down
DROP FUNCTION IF EXISTS prune_workspace_events(interval);
DROP TABLE IF EXISTS workspace_events;
DROP FUNCTION IF EXISTS workspace_events_notify();
