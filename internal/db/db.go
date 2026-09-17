package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

type Pool struct {
	pool *pgxpool.Pool
}

func Connect(ctx context.Context, databaseURL string) (*Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}
	cfg.MaxConns = 20

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return &Pool{pool: pool}, nil
}

func (p *Pool) Close() {
	p.pool.Close()
}

func (p *Pool) PgxPool() *pgxpool.Pool {
	return p.pool
}

func (p *Pool) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

func RunMigrations(ctx context.Context, databaseURL, migrationsDir string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open db for migrations: %w", err)
	}
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(2)

	locker, err := lock.NewPostgresSessionLocker(
		lock.WithLockTimeout(1, 60),
		lock.WithUnlockTimeout(1, 5),
	)
	if err != nil {
		return fmt.Errorf("create migration lock: %w", err)
	}
	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		db,
		os.DirFS(migrationsDir),
		goose.WithSessionLocker(locker),
	)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}
