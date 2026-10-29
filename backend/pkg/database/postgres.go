// Package database owns the PostgreSQL pool. It does not contain business
// logic, SQL queries, or transaction commits; those belong to the service
// layer. A service can call BeginTx and pass the resulting pgx.Tx to sqlc.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// DBTX is the interface generated sqlc code expects. Both *DB and pgx.Tx
// satisfy it, so the same generated queries work inside and outside a
// transaction.
type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type DB struct {
	pool   *pgxpool.Pool
	cfg    Config
	tracer trace.Tracer
}

var _ DBTX = (*DB)(nil)

func New(ctx context.Context, cfg Config) (*DB, error) {
	cfg = cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.HealthCheckPeriod
	poolConfig.ConnConfig.Tracer = newQueryTracer()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}

	db := &DB{pool: pool, cfg: cfg, tracer: otel.Tracer(instrumentationName)}
	pingContext, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	if err := pool.Ping(pingContext); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return db, nil
}

func NewFromURL(ctx context.Context, dsn string) (*DB, error) {
	return New(ctx, Config{URL: dsn})
}

func NewFromProvider(ctx context.Context, provider ConfigProvider, prefix string) (*DB, error) {
	return New(ctx, ConfigFrom(provider, prefix))
}

func (d *DB) Pool() *pgxpool.Pool {
	return d.pool
}

func (d *DB) Close() {
	if d.pool != nil {
		d.pool.Close()
	}
}

func (d *DB) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

func (d *DB) Health(ctx context.Context) error {
	pingContext, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return d.pool.Ping(pingContext)
}

func (d *DB) Stats() *pgxpool.Stat {
	return d.pool.Stat()
}

func (d *DB) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return d.pool.Exec(ctx, query, args...)
}

func (d *DB) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return d.pool.Query(ctx, query, args...)
}

func (d *DB) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return d.pool.QueryRow(ctx, query, args...)
}

func (d *DB) BeginTx(ctx context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	if d.tracer == nil {
		return d.pool.BeginTx(ctx, options)
	}

	ctx, span := d.tracer.Start(ctx, "db.begin",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system.name", "postgresql"),
			attribute.String("db.operation.name", "BEGIN"),
		),
	)
	defer span.End()

	tx, err := d.pool.BeginTx(ctx, options)
	if err != nil {
		recordDatabaseError(span, err)
		return nil, err
	}

	return tx, nil
}
