package migrations

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/samber/oops"
)

//go:embed *.sql
var migrationFS embed.FS

type Migration struct {
	Version int
	Name    string
	SQL     string
}

// All is built from the embedded *.sql files. Each filename must look like
// NNNN_name.sql (zero-padded version prefix, then the migration name); files
// are applied in ascending version order.
var All = loadAll()

func loadAll() []Migration {
	entries, err := migrationFS.ReadDir(".")
	if err != nil {
		panic(fmt.Sprintf("migrations: read embedded dir: %v", err))
	}
	var out []Migration
	seen := make(map[int]string)
	for _, e := range entries {
		filename := e.Name()
		if e.IsDir() || !strings.HasSuffix(filename, ".sql") {
			continue
		}
		stem := strings.TrimSuffix(filename, ".sql")
		versionStr, name, ok := strings.Cut(stem, "_")
		if !ok {
			panic(fmt.Sprintf("migrations: %q missing NNNN_name.sql version prefix", filename))
		}
		var version int
		if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil {
			panic(fmt.Sprintf("migrations: %q has a non-numeric version prefix", filename))
		}
		if other, dup := seen[version]; dup {
			panic(fmt.Sprintf("migrations: duplicate version %d in %q and %q", version, other, filename))
		}
		seen[version] = filename
		raw, err := migrationFS.ReadFile(filename)
		if err != nil {
			panic(fmt.Sprintf("migrations: read %q: %v", filename, err))
		}
		out = append(out, Migration{Version: version, Name: name, SQL: string(raw)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out
}

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
