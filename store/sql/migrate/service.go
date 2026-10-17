// Package migrate retains the existing setup API while delegating to one
// canonical schema creation path. There are no historical upgrade steps.
package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/viant/datly-studio/schema"
)

type Migration struct {
	Version              int
	Name, UpSQL, DownSQL string
}
type Service struct{}

func New() (*Service, error) { return &Service{}, nil }
func (*Service) Up(ctx context.Context, db *sql.DB) error {
	return schema.Initialize(ctx, db, "sqlite")
}
func (*Service) CurrentVersion(ctx context.Context, db *sql.DB) (int, error) {
	return schema.SQLiteVersion(ctx, db)
}
func (*Service) Migrations() []Migration {
	return []Migration{{Version: schema.CanonicalVersion, Name: "canonical_studio_schema"}}
}

// DownTo is retained only for explicit developer teardown, not application startup.
func (*Service) DownTo(ctx context.Context, db *sql.DB, target int) error {
	if target == 0 {
		return schema.DropSQLite(ctx, db)
	}
	if target != schema.CanonicalVersion {
		return fmt.Errorf("only the canonical Studio schema is supported")
	}
	return schema.Verify(ctx, db, "sqlite")
}
