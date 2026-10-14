package inventory_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Whitedragon115/Netopology-Backend/internal/authz"
	"github.com/Whitedragon115/Netopology-Backend/internal/inventory"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/testutil"
)

// Cancel after the database confirms COMMIT, before any subsequent response query.
type cancelOnCommit struct{ cancel context.CancelFunc }

func (tr *cancelOnCommit) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	return ctx
}

func (tr *cancelOnCommit) TraceQueryEnd(_ context.Context, _ *pgx.Conn, d pgx.TraceQueryEndData) {
	if d.Err == nil && d.CommandTag.String() == "COMMIT" {
		tr.cancel()
	}
}

func TestMutationReturnsSnapshotWhenContextCanceledAfterCommit(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	wsID := firstWorkspace(t, owner)
	user := userID(t, owner)
	ctx, cancel := context.WithCancel(authz.WithPrincipal(t.Context(), &authz.Principal{UserID: user, InstanceRole: authz.InstanceOwner}))
	defer cancel()
	cfg, err := pgxpool.ParseConfig(api.DB.AppURL)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.Tracer = &cancelOnCommit{cancel: cancel}
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	db := &pg.DB{Pool: pool}
	svc := inventory.NewService(db, authz.NewResolver(db))
	node, err := svc.CreateNode(ctx, wsID, inventory.NodeCreate{Name: "Committed snapshot", Category: "generic"})
	if err != nil {
		t.Fatalf("committed create reported failure: %v", err)
	}
	if ctx.Err() == nil || node.Name != "Committed snapshot" || node.Version != 1 {
		t.Fatalf("unexpected snapshot: %+v, context %v", node, ctx.Err())
	}
	owner.Do("GET", "/workspaces/"+wsID.String()+"/nodes/"+node.ID.String(), nil).Expect(t, 200)
}

func TestProjectionFailureRollsBackMutationAuditAndEvent(t *testing.T) {
	api := testutil.NewAPI(t)
	owner := api.Owner()
	wsID := firstWorkspace(t, owner)
	conn, err := pgx.Connect(t.Context(), api.DB.OwnerURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	// Inject a valid JSON array whose field value cannot be mapped to the wire type.
	_, err = conn.Exec(t.Context(), `CREATE FUNCTION corrupt_response() RETURNS trigger LANGUAGE plpgsql AS $$
	BEGIN NEW.properties = '[{"key":"bad","label":"Bad","value":{}}]'::jsonb; RETURN NEW; END $$;
	CREATE TRIGGER corrupt_response BEFORE INSERT ON nodes FOR EACH ROW EXECUTE FUNCTION corrupt_response()`)
	if err != nil {
		t.Fatal(err)
	}
	owner.Do("POST", "/workspaces/"+wsID.String()+"/nodes", map[string]any{"name": "Must roll back", "category": "generic"}).Expect(t, 500)
	var nodes, audits, events int
	if err := conn.QueryRow(t.Context(), `SELECT
	(SELECT count(*) FROM nodes WHERE workspace_id=$1),
	(SELECT count(*) FROM audit_log WHERE workspace_id=$1 AND action='node.create'),
	(SELECT count(*) FROM workspace_events WHERE workspace_id=$1 AND type='node.created')`, wsID).Scan(&nodes, &audits, &events); err != nil {
		t.Fatal(err)
	}
	if nodes != 0 || audits != 0 || events != 0 {
		t.Fatalf("projection failure committed partial state: %d %d %d", nodes, audits, events)
	}
}
