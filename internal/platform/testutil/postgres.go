// Package testutil gives integration tests a real, migrated, isolated PostgreSQL database and an
// HTTP client for the real API. No mocking the database (CONVENTIONS.md §3.2).
//
// How it works: NETOPOLOGY_TEST_ADMIN_URL points at a Postgres 18 superuser (the dev database from
// `task db:up`, or the CI service). Once per test process, a template database is created and
// migrated as a dedicated test owner role. Each test then gets its own copy
// (CREATE DATABASE ... TEMPLATE, ~50 ms) and connects as a dedicated NON-superuser app role, so
// row-level security is active exactly like production. The copy is dropped when the test ends.
//
// Without NETOPOLOGY_TEST_ADMIN_URL, integration tests are skipped (unit tests still run).
package testutil

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Whitedragon115/Netopology-Backend/db"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/migrate"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
)

const (
	testOwner    = "netopology_test_owner"
	testApp      = "netopology_test_app"
	testPassword = "netopology_test" // test-only roles on a dev/CI server, never production
)

var (
	templateOnce sync.Once
	templateName string
	templateErr  error
)

// DB is one isolated database for one test.
type DB struct {
	*pg.DB
	AppURL   string // NON-superuser (RLS applies) -- what the api uses
	OwnerURL string // schema owner -- for fixtures that must bypass the API
}

// Postgres returns a fresh migrated database for t, or skips t when no test server is configured.
func Postgres(t testing.TB) *DB {
	t.Helper()
	adminURL := os.Getenv("NETOPOLOGY_TEST_ADMIN_URL")
	if adminURL == "" {
		t.Skip("integration test: set NETOPOLOGY_TEST_ADMIN_URL (see docs/TESTING.md)")
	}
	ctx := context.Background()

	templateOnce.Do(func() { templateName, templateErr = ensureTemplate(ctx, adminURL) })
	if templateErr != nil {
		t.Fatalf("prepare template database: %v", templateErr)
	}

	name := "nt_" + randomHex(8)
	admin := connect(t, adminURL)
	if _, err := admin.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %s TEMPLATE %s OWNER %s`, name, templateName, testOwner)); err != nil {
		t.Fatalf("create test database: %v", err)
	}
	_ = admin.Close(ctx)
	t.Cleanup(func() {
		a, err := pgx.Connect(context.Background(), adminURL)
		if err == nil {
			_, _ = a.Exec(context.Background(), fmt.Sprintf(`DROP DATABASE IF EXISTS %s WITH (FORCE)`, name))
			_ = a.Close(context.Background())
		}
	})

	appURL := withUser(adminURL, testApp, name)
	d, err := pg.Open(ctx, appURL, 5)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(d.Close)
	return &DB{DB: d, AppURL: appURL, OwnerURL: withUser(adminURL, testOwner, name)}
}

// ensureTemplate creates the roles and one migrated template per migration set. The name contains
// a hash of the migrations, so changing a migration produces a new template automatically.
func ensureTemplate(ctx context.Context, adminURL string) (string, error) {
	hash, err := migrationsHash()
	if err != nil {
		return "", err
	}
	name := "nt_template_" + hash
	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		return "", fmt.Errorf("connect admin: %w", err)
	}
	defer admin.Close(ctx)

	// serialize with other test processes (go test runs packages in parallel)
	if _, err := admin.Exec(ctx, `SELECT pg_advisory_lock(hashtext('netopology.test.template'))`); err != nil {
		return "", err
	}
	defer admin.Exec(ctx, `SELECT pg_advisory_unlock(hashtext('netopology.test.template'))`) //nolint:errcheck

	var exists bool
	if err := admin.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, name).Scan(&exists); err != nil {
		return "", err
	}
	if exists {
		return name, nil
	}

	for _, stmt := range []string{
		fmt.Sprintf(`DO $$ BEGIN
		  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '%[1]s') THEN
		    CREATE ROLE %[1]s LOGIN NOSUPERUSER BYPASSRLS PASSWORD '%[3]s'; END IF;
		  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '%[2]s') THEN
		    CREATE ROLE %[2]s LOGIN NOSUPERUSER NOBYPASSRLS PASSWORD '%[3]s'; END IF;
		END $$`, testOwner, testApp, testPassword),
		fmt.Sprintf(`CREATE DATABASE %s OWNER %s`, name, testOwner),
	} {
		if _, err := admin.Exec(ctx, stmt); err != nil {
			return "", fmt.Errorf("%s: %w", stmt, err)
		}
	}

	// same privileges as db/init/10-roles.sh, inside the template
	tpl, err := pgx.Connect(ctx, withUser(adminURL, "", name))
	if err != nil {
		return "", err
	}
	for _, stmt := range []string{
		`REVOKE ALL ON SCHEMA public FROM PUBLIC`,
		fmt.Sprintf(`ALTER SCHEMA public OWNER TO %s`, testOwner),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA public TO %s`, testApp),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES FOR ROLE %s IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO %s`, testOwner, testApp),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES FOR ROLE %s IN SCHEMA public GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO %s`, testOwner, testApp),
		fmt.Sprintf(`ALTER DEFAULT PRIVILEGES FOR ROLE %s IN SCHEMA public GRANT EXECUTE ON FUNCTIONS TO %s`, testOwner, testApp),
	} {
		if _, err := tpl.Exec(ctx, stmt); err != nil {
			_ = tpl.Close(ctx)
			return "", fmt.Errorf("%s: %w", stmt, err)
		}
	}
	_ = tpl.Close(ctx)

	quiet := slog.New(slog.DiscardHandler)
	if err := migrate.Up(ctx, withUser(adminURL, testOwner, name), quiet); err != nil {
		return "", fmt.Errorf("migrate template: %w", err)
	}
	// a template must have no open connections when it is copied; mark it as a template
	if _, err := admin.Exec(ctx, fmt.Sprintf(`ALTER DATABASE %s WITH IS_TEMPLATE true`, name)); err != nil {
		return "", err
	}
	return name, nil
}

func migrationsHash() (string, error) {
	h := sha256.New()
	err := fs.WalkDir(db.Migrations, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(db.Migrations, path)
		if err != nil {
			return err
		}
		h.Write([]byte(path))
		h.Write(b)
		return nil
	})
	return hex.EncodeToString(h.Sum(nil))[:12], err
}

// withUser rewrites a connection URL to another user (test password) and/or database.
// An empty user keeps the admin credentials.
func withUser(raw, user, database string) string {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	if user != "" {
		u.User = url.UserPassword(user, testPassword)
	}
	u.Path = "/" + database
	return u.String()
}

func connect(t testing.TB, url string) *pgx.Conn {
	t.Helper()
	c, err := pgx.Connect(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	return c
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
