package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/timemachine-auto/timemachine/internal/config"
)

// New builds a configured pgx connection pool.
func New(ctx context.Context, cfg config.DBConfig) (*pgxpool.Pool, error) {
	pcfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	pcfg.MaxConns = cfg.MaxOpenConns
	pcfg.MinConns = cfg.MaxIdleConns
	pcfg.MaxConnIdleTime = cfg.MaxIdleTime
	pcfg.MaxConnLifetime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, pcfg)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}
