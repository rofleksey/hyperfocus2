// Package migrations holds the embedded SQL migrations and a small
// version-tracked runner. Each migration runs in its own transaction.
package migrations

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/samber/oops"
)

//go:embed 0001_init.sql
var initSQL string

// Migration is a single ordered, versioned schema change.
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// All is the ordered list of migrations.
var All = []Migration{
	{Version: 1, Name: "init_schema", SQL: initSQL},
}

// Run applies any pending migrations against the pool. It is idempotent.
func Run(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_version (
			version    INT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);`); err != nil {
		return oops.Wrap(err)
	}

	var maxVersion int
	row := pool.QueryRow(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_version;`)
	if err := row.Scan(&maxVersion); err != nil {
		return oops.Wrapf(err, "read schema_version")
	}

	for _, m := range All {
		if m.Version <= maxVersion {
			continue
		}
		if err := apply(ctx, pool, m); err != nil {
			return err
		}
		fmt.Printf("migrations: applied v%d %s\n", m.Version, m.Name)
	}
	return nil
}

func apply(ctx context.Context, pool *pgxpool.Pool, m Migration) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return oops.Wrapf(err, "begin tx for migration v%d", m.Version)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, m.SQL); err != nil {
		return oops.Wrapf(err, "exec migration v%d %q", m.Version, m.Name)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_version (version) VALUES ($1);`, m.Version); err != nil {
		return oops.Wrapf(err, "record migration v%d", m.Version)
	}
	if err := tx.Commit(ctx); err != nil {
		return oops.Wrapf(err, "commit migration v%d", m.Version)
	}
	return nil
}
