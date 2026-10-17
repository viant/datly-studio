package studioapi

import (
	"context"
	"database/sql"
	"github.com/viant/datly-studio/schema"
)

func ensureHostSchema(ctx context.Context, db *sql.DB, verify func(context.Context, *sql.DB) error) error {
	if verify != nil {
		return verify(ctx, db)
	}
	return ensureSchema(ctx, db)
}

// The configured store is created from one canonical schema when empty.
// An explicit embedding-host verifier retains ownership of external storage.
func ensureHostDatabaseSchema(ctx context.Context, db *sql.DB, driver string, verify func(context.Context, *sql.DB) error) error {
	if verify != nil {
		return verify(ctx, db)
	}
	return schema.Initialize(ctx, db, driver)
}
