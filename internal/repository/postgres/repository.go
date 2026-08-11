// Package postgres is the data-access adapter. A single Repository struct
// implements every narrow port interface declared by the usecases; consumers
// depend on those small interfaces (interface segregation), not on this type.
package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/samber/oops"

	"hyperfocus/internal/config"
)

// DBTX is the subset of *pgxpool.Pool / pgx.Tx used by repository methods.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Repository is the single, transaction-aware data-access implementation.
type Repository struct {
	pool *pgxpool.Pool
}

// New connects a pool and returns a Repository.
func New(ctx context.Context, cfg config.DB) (*Repository, error) {
	pcfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, oops.Wrap(err)
	}
	if cfg.MaxConns > 0 {
		pcfg.MaxConns = cfg.MaxConns
	}
	pcfg.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, pcfg)
	if err != nil {
		return nil, oops.Wrap(err)
	}
	return &Repository{pool: pool}, nil
}

// Pool exposes the underlying pool (used by the migration runner at startup).
func (r *Repository) Pool() *pgxpool.Pool { return r.pool }

// Close releases the pool's resources.
func (r *Repository) Close() {
	if r != nil && r.pool != nil {
		r.pool.Close()
	}
}

type txKey struct{}

// RunInTx runs f within a transaction. Methods called by f that use this
// repository will automatically use the transactional connection via context.
func (r *Repository) RunInTx(ctx context.Context, f func(ctx context.Context) error) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return oops.Wrap(err)
	}
	tctx := context.WithValue(ctx, txKey{}, tx)
	if err := f(tctx); err != nil {
		_ = tx.Rollback(context.Background())
		return err
	}
	if err := tx.Commit(tctx); err != nil {
		return oops.Wrap(err)
	}
	return nil
}

func (r *Repository) db(ctx context.Context) DBTX {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return r.pool
}
