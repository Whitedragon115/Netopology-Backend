// Package worker is the `netopology worker` process: background jobs on River (Postgres queue).
//
// Now: housekeeping (expired sessions, old live-update events).
// Later phases add here: health polling (C2/C3), workflow runs and the sandbox client (C4),
// plugin install/verify (C3). Every job that touches workspace data runs in pg.InWorkspace,
// exactly like a request, and writes its audit entries with actor_type 'system' or 'workflow'.
package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/config"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
)

// Run starts River and blocks until ctx is cancelled, then stops gracefully.
func Run(ctx context.Context, cfg *config.Config, db *pg.DB, log *slog.Logger) error {
	workers := river.NewWorkers()
	river.AddWorker(workers, &cleanupSessionsWorker{db: db, log: log})
	river.AddWorker(workers, &pruneEventsWorker{db: db, log: log})

	client, err := river.NewClient(riverpgxv5.New(db.Pool), &river.Config{
		Logger:  log,
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: cfg.WorkerConcurrency}},
		Workers: workers,
		PeriodicJobs: []*river.PeriodicJob{
			river.NewPeriodicJob(river.PeriodicInterval(time.Hour),
				func() (river.JobArgs, *river.InsertOpts) { return cleanupSessionsArgs{}, nil },
				&river.PeriodicJobOpts{RunOnStart: true}),
			river.NewPeriodicJob(river.PeriodicInterval(time.Hour),
				func() (river.JobArgs, *river.InsertOpts) { return pruneEventsArgs{}, nil },
				&river.PeriodicJobOpts{RunOnStart: true}),
		},
	})
	if err != nil {
		return fmt.Errorf("river client: %w", err)
	}
	if err := client.Start(ctx); err != nil {
		return fmt.Errorf("start river: %w", err)
	}
	log.InfoContext(ctx, "worker started", "concurrency", cfg.WorkerConcurrency)
	<-ctx.Done()

	stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return client.Stop(stopCtx)
}

type cleanupSessionsArgs struct{}

func (cleanupSessionsArgs) Kind() string { return "cleanup_sessions" }

type cleanupSessionsWorker struct {
	river.WorkerDefaults[cleanupSessionsArgs]
	db  *pg.DB
	log *slog.Logger
}

func (w *cleanupSessionsWorker) Work(ctx context.Context, _ *river.Job[cleanupSessionsArgs]) error {
	n, err := w.db.Q().DeleteExpiredSessions(ctx)
	if err != nil {
		return err
	}
	w.log.InfoContext(ctx, "expired sessions deleted", "count", n)
	return nil
}

type pruneEventsArgs struct{}

func (pruneEventsArgs) Kind() string { return "prune_workspace_events" }

type pruneEventsWorker struct {
	river.WorkerDefaults[pruneEventsArgs]
	db  *pg.DB
	log *slog.Logger
}

func (w *pruneEventsWorker) Work(ctx context.Context, _ *river.Job[pruneEventsArgs]) error {
	n, err := w.db.Q().PruneWorkspaceEvents(ctx, 24)
	if err != nil {
		return err
	}
	w.log.InfoContext(ctx, "old live-update events pruned", "count", n)
	return nil
}
