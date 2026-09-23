package migrate

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/viant/datly-studio/schema"
)

// Migration describes the canonical Studio schema snapshot. Domain DDL lives
// only in schema/schema.ddl.
type Migration struct {
	Version int
	Name    string
	UpSQL   string
	DownSQL string
}

type Service struct{}

func New() (*Service, error) { return &Service{}, nil }

func (s *Service) Up(ctx context.Context, db *sql.DB) error {
	current, err := schema.SQLiteVersion(ctx, db)
	if err != nil {
		return err
	}
	if current > schema.CanonicalVersion {
		return fmt.Errorf("database schema version %d is newer than supported version %d", current, schema.CanonicalVersion)
	}
	if current == schema.CanonicalVersion {
		return nil
	}
	if current == 0 {
		if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
			return fmt.Errorf("apply canonical schema: %w", err)
		}
		return schema.SetSQLiteVersion(ctx, db, schema.CanonicalVersion)
	}
	if current == 1 {
		if err := schema.AddSQLiteColumnFromCanonical(ctx, db, "reports", "namespace"); err != nil {
			return fmt.Errorf("add reports namespace: %w", err)
		}
		if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_reports_owner_namespace_updated ON reports(owner_id, namespace, updated_at DESC)`); err != nil {
			return fmt.Errorf("index reports namespace: %w", err)
		}
	}
	if current <= 2 {
		if err := schema.CreateSQLiteTableFromCanonical(ctx, db, "namespaces"); err != nil {
			return fmt.Errorf("create namespaces: %w", err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at)
SELECT owner_id,namespace,namespace,'active',1,MIN(updated_at),MAX(updated_at)
FROM reports GROUP BY owner_id,namespace`); err != nil {
			return fmt.Errorf("backfill namespaces: %w", err)
		}
	}
	if current <= 3 {
		exists, err := sqliteTableExists(ctx, db, "report_publications")
		if err != nil {
			return err
		}
		if exists {
			if err := schema.AddSQLiteColumnFromCanonical(ctx, db, "report_publications", "desired_version_no"); err != nil {
				return fmt.Errorf("add publication desired version: %w", err)
			}
			if _, err := db.ExecContext(ctx, `UPDATE report_publications SET desired_version_no=active_version_no WHERE desired_version_no IS NULL`); err != nil {
				return fmt.Errorf("backfill publication desired version: %w", err)
			}
		}
	}
	if current <= 4 {
		if err := schema.CreateSQLiteTableFromCanonical(ctx, db, "report_warmup_runs"); err != nil {
			return fmt.Errorf("create warmup runs: %w", err)
		}
	}
	if current <= 5 {
		if err := schema.CreateSQLiteTableFromCanonical(ctx, db, "bff_sessions"); err != nil {
			return fmt.Errorf("create BFF sessions: %w", err)
		}
	}
	if current <= 6 {
		if err := schema.CreateSQLiteTableFromCanonical(ctx, db, "report_publication_events"); err != nil {
			return fmt.Errorf("create publication events: %w", err)
		}
	}
	if current <= 7 {
		exists, err := sqliteTableExists(ctx, db, "report_acl")
		if err != nil {
			return err
		}
		if exists {
			if err := schema.AddSQLiteColumnFromCanonical(ctx, db, "report_acl", "etag"); err != nil {
				return fmt.Errorf("add report ACL etag: %w", err)
			}
		}
	}
	if current <= 8 {
		if err := schema.CreateSQLiteTableFromCanonical(ctx, db, "authorization_predicates"); err != nil {
			return fmt.Errorf("create authorization predicates: %w", err)
		}
	}
	if current == 9 {
		if err := schema.AddSQLiteColumnFromCanonical(ctx, db, "authorization_predicates", "sql_scope_json"); err != nil {
			return fmt.Errorf("add authorization predicate SQL scope: %w", err)
		}
	}
	if current <= 10 {
		for _, table := range []string{"resource_policy_heads", "resource_policy_revisions"} {
			if err := schema.CreateSQLiteTableFromCanonical(ctx, db, table); err != nil {
				return fmt.Errorf("create %s: %w", table, err)
			}
		}
	}
	return schema.SetSQLiteVersion(ctx, db, schema.CanonicalVersion)
}

func sqliteTableExists(ctx context.Context, db *sql.DB, table string) (bool, error) {
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count); err != nil {
		return false, err
	}
	return count == 1, nil
}

func (s *Service) DownTo(ctx context.Context, db *sql.DB, target int) error {
	if target < 0 || target > schema.CanonicalVersion {
		return fmt.Errorf("invalid target version: %d", target)
	}
	current, err := schema.SQLiteVersion(ctx, db)
	if err != nil {
		return err
	}
	if target >= current {
		return nil
	}
	if target == 0 {
		return schema.DropSQLite(ctx, db)
	}
	return nil
}

func (s *Service) CurrentVersion(ctx context.Context, db *sql.DB) (int, error) {
	return schema.SQLiteVersion(ctx, db)
}

func (s *Service) Migrations() []Migration {
	return []Migration{
		{Version: 1, Name: "canonical_studio_schema"},
		{Version: 2, Name: "report_business_namespace"},
		{Version: 3, Name: "governed_namespaces"},
		{Version: 4, Name: "staged_publication_versions"},
		{Version: 5, Name: "durable_warmup_runs"},
		{Version: 6, Name: "durable_bff_sessions"},
		{Version: 7, Name: "owner_scoped_publication_events"},
		{Version: 8, Name: "report_acl_row_concurrency"},
		{Version: 9, Name: "authorization_predicate_catalog"},
		{Version: 10, Name: "authorization_predicate_sql_scope"},
	}
}
