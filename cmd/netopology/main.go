// Command netopology is the single backend binary. The same image runs as different containers:
//
//	netopology api              HTTP API (REST + SSE). Runs migrations first if NETOPOLOGY_MIGRATE_ON_START.
//	netopology worker           background jobs (River)
//	netopology migrate [up]     apply database migrations (schema-owner role)
//	netopology migrate status   list applied / pending migrations
//	netopology openapi [-o f]   print the generated OpenAPI spec (YAML), or write it to file f
//	netopology healthcheck      exit 0 if the local api answers /readyz (Docker HEALTHCHECK; the image has no curl)
//	netopology version
//
// `export` (workspace export, phase C6) will be added here.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Whitedragon115/Netopology-Backend/internal/platform/config"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/events"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/logging"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/migrate"
	"github.com/Whitedragon115/Netopology-Backend/internal/platform/pg"
	"github.com/Whitedragon115/Netopology-Backend/internal/server"
	"github.com/Whitedragon115/Netopology-Backend/internal/worker"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "netopology:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cmd := "api"
	if len(args) > 0 {
		cmd = args[0]
	}
	switch cmd {
	case "version":
		fmt.Println(server.Version)
		return nil
	case "healthcheck":
		return healthcheck()
	case "openapi":
		out, err := server.OpenAPI().YAML()
		if err != nil {
			return err
		}
		if len(args) == 3 && args[1] == "-o" {
			return os.WriteFile(args[2], out, 0o644) //nolint:gosec // a public spec file
		}
		_, err = os.Stdout.Write(out)
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuration:\n%w", err)
	}
	log := logging.New(cfg.Env, cfg.LogLevel)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch cmd {
	case "migrate":
		return runMigrate(ctx, cfg, log, args[1:])
	case "api":
		return runAPI(ctx, cfg, log)
	case "worker":
		db, err := pg.Open(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
		if err != nil {
			return err
		}
		defer db.Close()
		return worker.Run(ctx, cfg, db, log)
	default:
		return fmt.Errorf("unknown command %q (api, worker, migrate, openapi, version)", cmd)
	}
}

func runMigrate(ctx context.Context, cfg *config.Config, log *slog.Logger, args []string) error {
	if cfg.MigrateDatabaseURL == "" {
		return errors.New("NETOPOLOGY_MIGRATE_DATABASE_URL is required for migrate")
	}
	if len(args) > 0 && args[0] == "status" {
		st, err := migrate.Status(ctx, cfg.MigrateDatabaseURL)
		if err != nil {
			return err
		}
		for _, s := range st {
			fmt.Printf("%-8s %05d  %s\n", s.State, s.Source.Version, s.Source.Path)
		}
		return nil
	}
	return migrate.Up(ctx, cfg.MigrateDatabaseURL, log)
}

func runAPI(ctx context.Context, cfg *config.Config, log *slog.Logger) error {
	if cfg.MigrateOnStart {
		if err := migrate.Up(ctx, cfg.MigrateDatabaseURL, log); err != nil {
			return err
		}
	}
	db, err := pg.Open(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		return err
	}
	defer db.Close()

	broker := events.NewBroker(db.Pool, log)
	go func() {
		if err := broker.Run(ctx); err != nil {
			log.Error("events broker stopped", "err", err)
		}
	}()

	handler, err := server.New(server.Deps{Config: cfg, DB: db, Log: log, Broker: broker})
	if err != nil {
		return err
	}
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
		// no WriteTimeout: SSE streams are long-lived; per-operation limits are set in handlers
	}
	errc := make(chan error, 1)
	go func() {
		log.Info("api listening", "addr", cfg.Addr, "public_url", cfg.PublicURL, "env", cfg.Env)
		errc <- srv.ListenAndServe()
	}()
	select {
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	log.Info("api shutting down")
	return srv.Shutdown(shutdownCtx)
}

// healthcheck needs no configuration besides the listen address, so it works in any container.
func healthcheck() error {
	addr := os.Getenv("NETOPOLOGY_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// host is fixed to loopback; only the port comes from our own env
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:"+port+"/readyz", nil) //nolint:gosec
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req) //nolint:gosec // see above
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("readyz: %s", res.Status)
	}
	return nil
}
