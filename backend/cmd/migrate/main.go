package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/timemachine-auto/timemachine/internal/config"
	"github.com/timemachine-auto/timemachine/internal/migrations"
	"github.com/timemachine-auto/timemachine/pkg/db"
)

func main() {
	direction := flag.String("direction", "up", "up | down")
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("svc", "migrate")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}
	pool, err := db.New(ctx, cfg.DB)
	if err != nil {
		log.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := ensureMigrationTable(ctx, pool); err != nil {
		log.Error("ensure table", "err", err)
		os.Exit(1)
	}

	files, err := migrations.FS.ReadDir(".")
	if err != nil {
		log.Error("read migrations", "err", err)
		os.Exit(1)
	}
	names := make([]string, 0, len(files))
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		names = append(names, f.Name())
	}
	sort.Strings(names)

	suffix := ".up.sql"
	if *direction == "down" {
		suffix = ".down.sql"
	}

	applied := 0
	for _, name := range names {
		if !strings.HasSuffix(name, suffix) {
			continue
		}
		if alreadyApplied(ctx, pool, name) {
			continue
		}
		path := name
		sql, err := migrations.FS.ReadFile(path)
		if err != nil {
			log.Error("read file", "name", name, "err", err)
			os.Exit(1)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			log.Error("exec migration FAILED", "name", name, "err", err)
			os.Exit(2)
		}
		if err := recordMigration(ctx, pool, name); err != nil {
			log.Error("record migration", "err", err)
			os.Exit(1)
		}
		log.Info("applied", "name", name)
		applied++
	}
	fmt.Printf("migrations %s: %d applied\n", *direction, applied)
}

func ensureMigrationTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		name text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`)
	return err
}

func alreadyApplied(ctx context.Context, pool *pgxpool.Pool, name string) bool {
	var n string
	err := pool.QueryRow(ctx, "SELECT name FROM schema_migrations WHERE name=$1", name).Scan(&n)
	return err == nil
}

func recordMigration(ctx context.Context, pool *pgxpool.Pool, name string) error {
	_, err := pool.Exec(ctx, "INSERT INTO schema_migrations (name) VALUES ($1) ON CONFLICT DO NOTHING", name)
	return err
}
