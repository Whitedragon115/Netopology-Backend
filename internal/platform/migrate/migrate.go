// Package migrate applies the schema: our goose migrations, then River's job-queue tables.
//
// It connects as the schema-owner role (NETOPOLOGY_MIGRATE_DATABASE_URL), never the app role.
// goose takes a Postgres session advisory lock, so api and worker starting together cannot race.
package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver for goose
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/Whitedragon115/Netopology-Backend/db"
)

// Up applies every pending migration. Safe to call on every start.
func Up(ctx context.Context, ownerURL string, log *slog.Logger) error {
	sqlDB, err := sql.Open("pgx", ownerURL)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer sqlDB.Close()

	migrations, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		return err
	}
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return err
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations, goose.WithSessionLocker(locker))
	if err != nil {
		return fmt.Errorf("goose: %w", err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	for _, r := range results {
		log.InfoContext(ctx, "migration applied", "version", r.Source.Version, "file", r.Source.Path, "duration", r.Duration)
	}

	pool, err := pgxpool.New(ctx, ownerURL)
	if err != nil {
		return fmt.Errorf("connect for river: %w", err)
	}
	defer pool.Close()
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return err
	}
	res, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	if err != nil {
		return fmt.Errorf("river migrations: %w", err)
	}
	for _, v := range res.Versions {
		log.InfoContext(ctx, "river migration applied", "version", v.Version)
	}
	return nil
}

// Status prints applied/pending migrations (used by `netopology migrate status`).
func Status(ctx context.Context, ownerURL string) ([]*goose.MigrationStatus, error) {
	sqlDB, err := sql.Open("pgx", ownerURL)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()
	migrations, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		return nil, err
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations)
	if err != nil {
		return nil, err
	}
	return provider.Status(ctx)
}
