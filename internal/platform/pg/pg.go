// Package pg owns the PostgreSQL connection pool and the transaction helpers every store uses.
//
// The two rules this package exists to enforce (docs/ARCHITECTURE.md §5):
//
//  1. Anything touching a workspace-owned table runs inside InWorkspace, which sets
//     app.workspace_id for the transaction so row-level security hides every other workspace.
//     The SQL must ALSO filter by workspace_id: RLS is the backstop, not the plan.
//  2. The pool connects as the non-superuser app role. Never point NETOPOLOGY_DATABASE_URL at
//     a superuser or BYPASSRLS role; Open refuses to start if it detects one.
package pg

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg/dbgen"
)

// DB is shared by all services. It is safe for concurrent use.
type DB struct {
	Pool *pgxpool.Pool
}

// Open connects, pings, and checks that the role is subject to RLS.
func Open(ctx context.Context, url string, maxConns int32) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	cfg.MaxConns = maxConns
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	var bypass bool
	if err := pool.QueryRow(ctx,
		`SELECT rolsuper OR rolbypassrls FROM pg_roles WHERE rolname = current_user`).Scan(&bypass); err != nil {
		pool.Close()
		return nil, fmt.Errorf("check role: %w", err)
	}
	if bypass {
		pool.Close()
		return nil, errors.New("NETOPOLOGY_DATABASE_URL uses a superuser or BYPASSRLS role; use the app role (netopology_app)")
	}
	return &DB{Pool: pool}, nil
}

func (d *DB) Close() { d.Pool.Close() }

// Q returns queries that run outside a transaction. Only for single reads of tables WITHOUT
// RLS (users, sessions, workspaces, workspace_members, share_links, instance_settings).
func (d *DB) Q() *dbgen.Queries { return dbgen.New(d.Pool) }

// Tx runs fn in a transaction that is NOT scoped to a workspace. Use it for the access-control
// tables listed on Q. Workspace-owned tables return no rows here (RLS), by design.
func (d *DB) Tx(ctx context.Context, fn func(q *dbgen.Queries) error) error {
	return pgx.BeginTxFunc(ctx, d.Pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return fn(dbgen.New(tx))
	})
}

// InWorkspace runs fn in a transaction scoped to one workspace (sets app.workspace_id, which
// every RLS policy compares against). All reads and writes of workspace content go through here,
// together with their audit entry and event (same transaction: all or nothing).
func (d *DB) InWorkspace(ctx context.Context, workspaceID uuid.UUID, fn func(q *dbgen.Queries) error) error {
	if workspaceID == uuid.Nil {
		return errors.New("pg.InWorkspace: empty workspace id")
	}
	return pgx.BeginTxFunc(ctx, d.Pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT set_config('app.workspace_id', $1, true)`, workspaceID.String()); err != nil {
			return fmt.Errorf("set workspace scope: %w", err)
		}
		return fn(dbgen.New(tx))
	})
}
