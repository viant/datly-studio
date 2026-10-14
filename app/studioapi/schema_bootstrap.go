package studioapi

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/viant/datly-studio/schema"
)

func ensureHostSchema(ctx context.Context, db *sql.DB, verify func(context.Context, *sql.DB) error) error {
	if verify != nil {
		return verify(ctx, db)
	}
	return ensureSchema(ctx, db)
}

// MySQL storage is supplied by operator-managed SQL migrations. Startup checks
// it without creating or changing tables; SQLite keeps its existing bootstrap.
func ensureHostDatabaseSchema(ctx context.Context, db *sql.DB, driver string, verify func(context.Context, *sql.DB) error) error {
	if verify != nil {
		return verify(ctx, db)
	}
	if driver == "sqlite" {
		return ensureSchema(ctx, db)
	}
	if driver != "mysql" {
		return fmt.Errorf("unsupported Studio catalog driver")
	}
	var count, minimum, maximum int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*),MIN(version),MAX(version) FROM schema_version").Scan(&count, &minimum, &maximum); err != nil || count != 1 || minimum != schema.CanonicalVersion || maximum != schema.CanonicalVersion {
		return fmt.Errorf("Studio MySQL schema migrations must be applied before startup")
	}
	for _, table := range []string{"namespaces", "connectors", "components", "component_versions", "namespace_skills", "bff_sessions"} {
		rows, err := db.QueryContext(ctx, "SELECT * FROM "+table+" WHERE 1=0")
		if err != nil {
			return fmt.Errorf("Studio MySQL storage is incomplete")
		}
		if err = rows.Close(); err != nil {
			return err
		}
	}
	return nil
}
