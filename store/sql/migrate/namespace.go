package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/schema"
)

// migrateNamespaceOwnership preserves existing data while assigning namespace
// identity to roots and descendants. Namespace visibility starts private.
func migrateNamespaceOwnership(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	namespaceTable, err := sqliteTableExists(ctx, tx, "namespaces")
	if err != nil {
		return err
	}
	if !namespaceTable {
		if err = schema.CreateSQLiteTableFromCanonical(ctx, tx, "namespaces"); err != nil {
			return err
		}
	}
	componentTable, err := sqliteTableExists(ctx, tx, "components")
	if err != nil {
		return err
	}

	for _, column := range []string{"namespace_id", "visibility", "allowed_roles_json", "mcp_enabled", "mcp_port"} {
		exists, err := sqliteColumnExists(ctx, tx, "namespaces", column)
		if err != nil {
			return err
		}
		if !exists {
			if err = schema.AddSQLiteColumnFromCanonical(ctx, tx, "namespaces", column); err != nil {
				return err
			}
		}
	}
	// Legacy databases can contain component namespaces without a catalog row.
	// Preserve those keys with private visibility before assigning stable IDs.
	if componentTable {
		hasOwner, err := sqliteColumnExists(ctx, tx, "components", "owner_id")
		if err != nil {
			return err
		}
		hasName, err := sqliteColumnExists(ctx, tx, "components", "namespace")
		if err != nil {
			return err
		}
		if hasOwner && hasName {
			_, err = tx.ExecContext(ctx, `INSERT INTO namespaces
				(owner_id,name,title,status,etag,created_at,updated_at)
				SELECT DISTINCT c.owner_id,c.namespace,c.namespace,'active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
				FROM components c
				WHERE NOT EXISTS (SELECT 1 FROM namespaces n WHERE n.owner_id=c.owner_id AND n.name=c.namespace)`)
			if err != nil {
				return fmt.Errorf("preserve component namespaces: %w", err)
			}
		}
	}
	rows, err := tx.QueryContext(ctx, "SELECT owner_id,name FROM namespaces")
	if err != nil {
		return err
	}
	type key struct{ owner, name string }
	var keys []key
	for rows.Next() {
		var item key
		if err = rows.Scan(&item.owner, &item.name); err != nil {
			rows.Close()
			return err
		}
		keys = append(keys, item)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, item := range keys {
		if _, err = tx.ExecContext(ctx, "UPDATE namespaces SET namespace_id=? WHERE owner_id=? AND name=?", namespaceaccess.ID(item.owner, item.name), item.owner, item.name); err != nil {
			return err
		}
	}
	tables, err := tx.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name NOT IN ('namespaces','connectors','schema_version','sqlx_sequence_reservations')")
	if err != nil {
		return err
	}
	var names []string
	for tables.Next() {
		var name string
		if err = tables.Scan(&name); err != nil {
			tables.Close()
			return err
		}
		names = append(names, name)
	}
	if err = tables.Err(); err != nil {
		tables.Close()
		return err
	}
	tables.Close()
	for _, table := range names {
		present, err := sqliteColumnExists(ctx, tx, table, "namespace_id")
		if err != nil {
			return err
		}
		if !present {
			if err = schema.AddSQLiteColumnFromCanonical(ctx, tx, table, "namespace_id"); err != nil {
				return fmt.Errorf("scope %s: %w", table, err)
			}
		}
		switch table {
		case "components":
			hasOwner, ownerErr := sqliteColumnExists(ctx, tx, table, "owner_id")
			if ownerErr != nil {
				return ownerErr
			}
			hasName, nameErr := sqliteColumnExists(ctx, tx, table, "namespace")
			if nameErr != nil {
				return nameErr
			}
			if !hasOwner || !hasName {
				continue
			}
			_, err = tx.ExecContext(ctx, `UPDATE components SET namespace_id=(SELECT namespace_id FROM namespaces n WHERE n.owner_id=components.owner_id AND n.name=components.namespace)`)
		default:
			hasReport, lookupErr := sqliteColumnExists(ctx, tx, table, "report_id")
			if lookupErr != nil {
				return lookupErr
			}
			if hasReport && componentTable {
				_, err = tx.ExecContext(ctx, `UPDATE "`+table+`" SET namespace_id=COALESCE((SELECT namespace_id FROM components c WHERE c.id="`+table+`".report_id),'')`)
			}
		}
		if err != nil {
			return err
		}
	}
	if err = assignLegacyGenerationOwnership(ctx, tx); err != nil {
		return err
	}
	if err := rebuildNamespaceClaimKeys(ctx, tx); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM schema_version"); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO schema_version(version) VALUES (?)", schema.CanonicalVersion); err != nil {
		return err
	}
	return tx.Commit()
}
