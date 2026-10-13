package events

import (
	"context"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Broker fans out pg_notify("netopology_events") to the SSE connections of this process.
//
// TODO(WP-11): implement Run:
//   - acquire one dedicated connection from the pool (pool.Acquire, then Hijack it so it never
//     returns to the pool), run `LISTEN netopology_events`;
//   - loop on conn.WaitForNotification(ctx); payload is "<workspace_id>:<event_id>"; parse it and
//     publish a Ref to every subscriber of that workspace (non-blocking send; if a subscriber's
//     buffer is full, close it: the client reconnects with Last-Event-ID and catches up);
//   - on connection loss: log, back off (1s..30s), reconnect, and send every subscriber a
//     "resync" signal (close it) because notifications may have been missed;
//   - return when ctx is cancelled.
//
// Unit-test with a real Postgres (testutil.Postgres): insert a workspace_events row in a
// committed transaction and assert the subscriber receives the Ref; a rolled-back insert must
// deliver nothing.
type Broker struct {
	pool *pgxpool.Pool
	log  *slog.Logger

	mu   sync.Mutex
	subs map[uuid.UUID]map[*Subscription]struct{} // by workspace
}

// Subscription is one open SSE connection.
type Subscription struct {
	WorkspaceID uuid.UUID
	C           chan Ref // closed when the subscription ends or must resync
	broker      *Broker
}

func NewBroker(pool *pgxpool.Pool, log *slog.Logger) *Broker {
	return &Broker{pool: pool, log: log, subs: map[uuid.UUID]map[*Subscription]struct{}{}}
}

// Run listens until ctx is cancelled. Started once by the api process.
func (b *Broker) Run(ctx context.Context) error {
	b.log.InfoContext(ctx, "events broker: not implemented yet (WP-11)")
	<-ctx.Done()
	return nil
}

// Subscribe registers a connection for one workspace. Call Close when the client goes away.
func (b *Broker) Subscribe(workspaceID uuid.UUID) *Subscription {
	s := &Subscription{WorkspaceID: workspaceID, C: make(chan Ref, 64), broker: b}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.subs[workspaceID] == nil {
		b.subs[workspaceID] = map[*Subscription]struct{}{}
	}
	b.subs[workspaceID][s] = struct{}{}
	return s
}

func (s *Subscription) Close() {
	b := s.broker
	b.mu.Lock()
	defer b.mu.Unlock()
	if set, ok := b.subs[s.WorkspaceID]; ok {
		if _, ok := set[s]; ok {
			delete(set, s)
			close(s.C)
		}
		if len(set) == 0 {
			delete(b.subs, s.WorkspaceID)
		}
	}
}
