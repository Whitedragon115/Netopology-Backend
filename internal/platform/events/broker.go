package events

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

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

	mu    sync.Mutex
	subs  map[uuid.UUID]map[*Subscription]struct{} // by workspace
	ready chan struct{}
	once  sync.Once
}

// Subscription is one open SSE connection.
type Subscription struct {
	WorkspaceID uuid.UUID
	C           chan Ref // closed when the subscription ends or must resync
	broker      *Broker
}

func NewBroker(pool *pgxpool.Pool, log *slog.Logger) *Broker {
	return &Broker{pool: pool, log: log, subs: map[uuid.UUID]map[*Subscription]struct{}{}, ready: make(chan struct{})}
}

// Ready closes after the first LISTEN succeeds; useful for startup and integration tests.
func (b *Broker) Ready() <-chan struct{} { return b.ready }

// Run listens until ctx is cancelled. Started once by the api process.
func (b *Broker) Run(ctx context.Context) error {
	defer b.resync()
	delay := time.Second
	for ctx.Err() == nil {
		pooled, err := b.pool.Acquire(ctx)
		if err == nil {
			conn := pooled.Hijack()
			_, err = conn.Exec(ctx, "LISTEN netopology_events")
			if err == nil {
				b.once.Do(func() { close(b.ready) })
				delay = time.Second
				for ctx.Err() == nil {
					n, waitErr := conn.WaitForNotification(ctx)
					if waitErr != nil {
						err = waitErr
						break
					}
					workspace, event, ok := strings.Cut(n.Payload, ":")
					if !ok {
						continue
					}
					wsID, parseErr := uuid.Parse(workspace)
					if parseErr != nil {
						continue
					}
					eventID, parseErr := uuid.Parse(event)
					if parseErr != nil {
						continue
					}
					b.publish(Ref{WorkspaceID: wsID, EventID: eventID})
				}
			}
			closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			_ = conn.Close(closeCtx)
			cancel()
		}
		b.resync() // a notification may have been lost: reconnect clients with their cursor
		if ctx.Err() != nil {
			break
		}
		b.log.WarnContext(ctx, "event listener reconnecting", "err", err)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
		if delay < 30*time.Second {
			delay = min(delay*2, 30*time.Second)
		}
	}
	return nil
}

func (b *Broker) publish(ref Ref) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for s := range b.subs[ref.WorkspaceID] {
		select {
		case s.C <- ref:
		default:
			delete(b.subs[ref.WorkspaceID], s)
			close(s.C)
		}
	}
	if len(b.subs[ref.WorkspaceID]) == 0 {
		delete(b.subs, ref.WorkspaceID)
	}
}

func (b *Broker) resync() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, set := range b.subs {
		for s := range set {
			close(s.C)
		}
	}
	b.subs = map[uuid.UUID]map[*Subscription]struct{}{}
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
