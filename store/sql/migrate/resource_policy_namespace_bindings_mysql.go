package migrate

import (
	"context"
	"database/sql"
	"fmt"

	policySchema "github.com/viant/authz/component/schema"
	"github.com/viant/datly-studio/schema"
)

// MigratePolicyNamespaceBindingsMySQL is a Studio-owned migration for existing
// databases that already have canonical shared policy tables with legacy
// namespace columns. Authz supplies fresh policy DDL only; Studio copies its
// namespace metadata into the sidecar binding table and never converts shared
// table layouts.
func MigratePolicyNamespaceBindingsMySQL(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("MySQL database is required")
	}
	if err := policySchema.CreatePolicies(ctx, db, "mysql"); err != nil {
		return err
	}
	ddl, err := schema.MySQLTableStatement("resource_policy_namespace_bindings")
	if err != nil {
		return err
	}
	bindingTableExists, err := mysqlTableExists(ctx, db, "resource_policy_namespace_bindings")
	if err != nil {
		return err
	}
	if !bindingTableExists {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("create Studio policy namespace binding table: %w", err)
		}
	}
	for _, column := range []string{"tenant_id", "resource_kind", "resource_id", "resource_version", "policy_revision", "namespace_id", "created_at", "created_by", "updated_at", "updated_by"} {
		exists, err := mysqlColumnExists(ctx, db, "resource_policy_namespace_bindings", column)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("Studio policy namespace binding table lacks %s", column)
		}
	}
	if err := migrateByteExactPolicyBindingKeysMySQL(ctx, db); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, source := range []struct {
		table          string
		policyRevision string
	}{{table: "resource_policies", policyRevision: "0"}, {table: "resource_policy_revisions", policyRevision: "src.revision"}} {
		hasNamespace, err := mysqlColumnExists(ctx, tx, source.table, "namespace_id")
		if err != nil {
			return err
		}
		if !hasNamespace {
			continue
		}
		for _, column := range []string{"created_at", "created_by", "updated_at", "updated_by"} {
			hasColumn, err := mysqlColumnExists(ctx, tx, source.table, column)
			if err != nil {
				return err
			}
			if !hasColumn {
				return fmt.Errorf("shared policy table %s lacks audit column %s", source.table, column)
			}
		}
		conflict := `SELECT COUNT(*) FROM ` + source.table + ` src JOIN resource_policy_namespace_bindings b
			ON BINARY b.tenant_id=BINARY src.tenant_id AND BINARY b.resource_kind=BINARY src.resource_kind AND BINARY b.resource_id=BINARY src.resource_id
			AND BINARY b.resource_version=BINARY src.resource_version AND b.policy_revision=` + source.policyRevision + `
			WHERE COALESCE(src.namespace_id,'')<>'' AND BINARY b.namespace_id<>BINARY src.namespace_id`
		var conflicts int
		if err := tx.QueryRowContext(ctx, conflict).Scan(&conflicts); err != nil {
			return err
		}
		if conflicts != 0 {
			return fmt.Errorf("conflicting Studio policy namespace binding in %s", source.table)
		}
		insert := `INSERT IGNORE INTO resource_policy_namespace_bindings
			(tenant_id,resource_kind,resource_id,resource_version,policy_revision,namespace_id,created_at,created_by,updated_at,updated_by)
			SELECT src.tenant_id,src.resource_kind,src.resource_id,src.resource_version,` + source.policyRevision + `,
				src.namespace_id,src.created_at,src.created_by,src.updated_at,src.updated_by FROM ` + source.table + ` src
			WHERE COALESCE(src.namespace_id,'')<>''`
		if _, err := tx.ExecContext(ctx, insert); err != nil {
			return fmt.Errorf("copy Studio namespace bindings from %s: %w", source.table, err)
		}
		verify := `SELECT COUNT(*) FROM ` + source.table + ` src WHERE COALESCE(src.namespace_id,'')<>'' AND NOT EXISTS (
			SELECT 1 FROM resource_policy_namespace_bindings b WHERE BINARY b.tenant_id=BINARY src.tenant_id
			AND BINARY b.resource_kind=BINARY src.resource_kind AND BINARY b.resource_id=BINARY src.resource_id AND BINARY b.resource_version=BINARY src.resource_version
			AND b.policy_revision=` + source.policyRevision + ` AND b.namespace_id=src.namespace_id)`
		var missing int
		if err := tx.QueryRowContext(ctx, verify).Scan(&missing); err != nil {
			return err
		}
		if missing != 0 {
			return fmt.Errorf("Studio namespace binding copy from %s left %d records unmapped", source.table, missing)
		}
	}
	return tx.Commit()
}

func migrateByteExactPolicyBindingKeysMySQL(ctx context.Context, db *sql.DB) error {
	wantLengths := []struct {
		name         string
		chars, bytes int64
	}{{"tenant_id", 128, 512}, {"resource_kind", 64, 256}, {"resource_id", 200, 800}, {"resource_version", 64, 256}}
	for _, item := range wantLengths {
		var dataType string
		var maxLength sql.NullInt64
		var charset sql.NullString
		if err := db.QueryRowContext(ctx, `SELECT data_type,character_maximum_length,character_set_name
			FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='resource_policy_namespace_bindings' AND column_name=?`, item.name).
			Scan(&dataType, &maxLength, &charset); err != nil {
			return err
		}
		legacy := dataType == "varchar" && maxLength.Valid && maxLength.Int64 == item.chars && charset.Valid && charset.String == "utf8mb4"
		exact := dataType == "varbinary" && maxLength.Valid && maxLength.Int64 == item.bytes && !charset.Valid
		if !legacy && !exact {
			return fmt.Errorf("incompatible Studio policy namespace key column %s", item.name)
		}
		var oversized int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM resource_policy_namespace_bindings WHERE OCTET_LENGTH("+item.name+")>?", item.bytes).Scan(&oversized); err != nil {
			return err
		}
		if oversized != 0 {
			return fmt.Errorf("Studio policy namespace key %s contains values beyond the byte-exact key capacity", item.name)
		}
	}
	rows, err := db.QueryContext(ctx, `SELECT seq_in_index,column_name,sub_part FROM information_schema.statistics
		WHERE table_schema=DATABASE() AND table_name='resource_policy_namespace_bindings' AND index_name='PRIMARY' ORDER BY seq_in_index`)
	if err != nil {
		return err
	}
	want := []string{"tenant_id", "resource_kind", "resource_id", "resource_version", "policy_revision"}
	index := 0
	for rows.Next() {
		var sequence int
		var column string
		var prefix sql.NullInt64
		if err := rows.Scan(&sequence, &column, &prefix); err != nil {
			rows.Close()
			return err
		}
		if index >= len(want) || sequence != index+1 || column != want[index] || prefix.Valid {
			rows.Close()
			return fmt.Errorf("incompatible Studio policy namespace binding primary key")
		}
		index++
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if index != len(want) {
		return fmt.Errorf("incompatible Studio policy namespace binding primary key")
	}
	var duplicates int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM (
		SELECT 1 FROM resource_policy_namespace_bindings
		GROUP BY BINARY tenant_id,BINARY resource_kind,BINARY resource_id,BINARY resource_version,policy_revision
		HAVING COUNT(*)>1) exact_keys`).Scan(&duplicates); err != nil {
		return err
	}
	if duplicates != 0 {
		return fmt.Errorf("Studio policy namespace binding contains duplicate exact keys")
	}
	for _, item := range wantLengths {
		var dataType string
		var length int64
		if err := db.QueryRowContext(ctx, `SELECT data_type,character_maximum_length FROM information_schema.columns
			WHERE table_schema=DATABASE() AND table_name='resource_policy_namespace_bindings' AND column_name=?`, item.name).Scan(&dataType, &length); err != nil {
			return err
		}
		if dataType == "varbinary" && length == item.bytes {
			continue
		}
		definition := fmt.Sprintf("%s VARBINARY(%d) NOT NULL", item.name, item.bytes)
		if _, err := db.ExecContext(ctx, "ALTER TABLE resource_policy_namespace_bindings MODIFY "+definition); err != nil {
			return fmt.Errorf("apply byte-exact Studio policy binding key %s: %w", item.name, err)
		}
	}
	return nil
}

func mysqlColumnExists(ctx context.Context, db interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, table, column string) (bool, error) {
	var count int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name=?`, table, column).Scan(&count)
	return count == 1, err
}

func mysqlTableExists(ctx context.Context, db interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, table string) (bool, error) {
	var count int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name=?`, table).Scan(&count)
	return count == 1, err
}
