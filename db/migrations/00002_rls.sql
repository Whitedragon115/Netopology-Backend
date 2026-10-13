-- Row-Level Security: a second safety net for workspace isolation.
-- The application must ALSO scope every query by workspace_id (CONVENTIONS.md §3.1 rule 3).
--
-- How it works: per transaction, the API runs
--   SELECT set_config('app.workspace_id', '<uuid>', true);
-- and these policies hide every row of every other workspace.
--
-- IMPORTANT: superusers and roles with BYPASSRLS ignore these policies.
-- The app must connect as a normal role (see docs/design/data-model.md §6).
--
-- Not covered (they ARE the access-control layer and are queried before a workspace is chosen):
--   users, sessions, instance_settings, workspaces, workspace_members, share_links, audit_log

-- +goose Up

-- +goose StatementBegin
CREATE FUNCTION current_workspace_id() RETURNS uuid
LANGUAGE sql STABLE AS $$
  SELECT NULLIF(current_setting('app.workspace_id', true), '')::uuid
$$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
  t text;
BEGIN
  FOREACH t IN ARRAY ARRAY['areas', 'nodes', 'links', 'notes', 'canvases', 'canvas_grants', 'placements', 'canvas_shapes']
  LOOP
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
    EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
    EXECUTE format(
      'CREATE POLICY workspace_isolation ON %I USING (workspace_id = current_workspace_id()) WITH CHECK (workspace_id = current_workspace_id())',
      t);
  END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
DECLARE
  t text;
BEGIN
  FOREACH t IN ARRAY ARRAY['areas', 'nodes', 'links', 'notes', 'canvases', 'canvas_grants', 'placements', 'canvas_shapes']
  LOOP
    EXECUTE format('DROP POLICY IF EXISTS workspace_isolation ON %I', t);
    EXECUTE format('ALTER TABLE %I NO FORCE ROW LEVEL SECURITY', t);
    EXECUTE format('ALTER TABLE %I DISABLE ROW LEVEL SECURITY', t);
  END LOOP;
END $$;
-- +goose StatementEnd
DROP FUNCTION IF EXISTS current_workspace_id();
