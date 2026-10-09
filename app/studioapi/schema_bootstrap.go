package studioapi

import (
	"context"
	"database/sql"
)

func ensureHostSchema(ctx context.Context, db *sql.DB, verify func(context.Context, *sql.DB) error) error {
	if verify != nil {
		return verify(ctx, db)
	}
	return ensureSchema(ctx, db)
}
