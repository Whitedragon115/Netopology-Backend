package events_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/events"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/testutil"
)

func TestNotificationsFollowCommit(t *testing.T) {
	db := testutil.Postgres(t)
	ws, err := db.Q().CreateWorkspace(t.Context(), dbgen.CreateWorkspaceParams{Name: "Events", Slug: "events"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	broker := events.NewBroker(db.Pool, slog.New(slog.DiscardHandler))
	done := make(chan struct{})
	go func() { defer close(done); _ = broker.Run(ctx) }()
	t.Cleanup(func() { cancel(); <-done })
	select {
	case <-broker.Ready():
	case <-time.After(5 * time.Second):
		t.Fatal("LISTEN unavailable")
	}
	sub := broker.Subscribe(ws.ID)
	defer sub.Close()
	err = db.InWorkspace(ctx, ws.ID, func(q *dbgen.Queries) error {
		return events.Emit(ctx, q, events.Change{WorkspaceID: ws.ID, Type: events.NodeUpdated})
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case ref := <-sub.C:
		if ref.WorkspaceID != ws.ID {
			t.Fatal("wrong workspace")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("commit not delivered")
	}
	rollback := errors.New("rollback")
	err = db.InWorkspace(ctx, ws.ID, func(q *dbgen.Queries) error {
		if err := events.Emit(ctx, q, events.Change{WorkspaceID: ws.ID, Type: events.NodeUpdated}); err != nil {
			return err
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
	select {
	case <-sub.C:
		t.Fatal("rollback emitted a notification")
	case <-time.After(150 * time.Millisecond):
	}
}
