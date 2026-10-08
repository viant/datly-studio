package migrate

import (
	"context"
	"database/sql"
)

func migrateLegacyGenerationOwnership(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := assignLegacyGenerationOwnership(ctx, tx); err != nil {
		return err
	}
	if err := rebuildNamespaceClaimKeys(ctx, tx); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE schema_version SET version=?`, 19); err != nil {
		return err
	}
	return tx.Commit()
}

// Publication references must account for every component in the generation.
// Mixed, incomplete, orphaned and unreferenced history stays unassigned rather
// than becoming visible in an arbitrarily chosen namespace.
func assignLegacyGenerationOwnership(ctx context.Context, tx *sql.Tx) error {
	for table, columns := range map[string][]string{
		"runtime_generations":    {"namespace_id", "generation_no", "report_count"},
		"component_publications": {"report_id", "active_generation", "desired_generation"},
		"components":             {"id", "namespace_id"},
	} {
		present, err := sqliteTableExists(ctx, tx, table)
		if err != nil {
			return err
		}
		if !present {
			return nil
		}
		for _, column := range columns {
			present, err := sqliteColumnExists(ctx, tx, table, column)
			if err != nil {
				return err
			}
			if !present {
				return nil
			}
		}
	}
	_, err := tx.ExecContext(ctx, `UPDATE runtime_generations AS g
SET namespace_id=(SELECT MIN(c.namespace_id) FROM component_publications p JOIN components c ON c.id=p.report_id
    WHERE p.active_generation=g.generation_no OR p.desired_generation=g.generation_no)
WHERE g.namespace_id='' AND g.report_count>0
AND (SELECT COUNT(DISTINCT p.report_id) FROM component_publications p
    WHERE p.active_generation=g.generation_no OR p.desired_generation=g.generation_no)=g.report_count
AND (SELECT COUNT(DISTINCT c.id) FROM component_publications p JOIN components c ON c.id=p.report_id
    WHERE (p.active_generation=g.generation_no OR p.desired_generation=g.generation_no) AND c.namespace_id<>'')=g.report_count
AND (SELECT COUNT(DISTINCT c.namespace_id) FROM component_publications p JOIN components c ON c.id=p.report_id
    WHERE p.active_generation=g.generation_no OR p.desired_generation=g.generation_no)=1`)
	return err
}
