package migrate

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/viant/datly-studio/schema"
)

// This opt-in test covers the unrelated Studio-owned authorization-predicate
// key migration on a disposable MySQL database. Authz policy tables are not
// converted or versioned here; their fresh canonical DDL is tested by schema.
func TestAuthorizationPredicateKeyMySQLDisposable(t *testing.T) {
	rawDSN := strings.TrimSpace(os.Getenv("STUDIO_POLICY_SCHEMA_MYSQL_DSN"))
	if rawDSN == "" {
		t.Skip("set STUDIO_POLICY_SCHEMA_MYSQL_DSN to an isolated studio_schema_test database on 127.0.0.1:23309")
	}
	cfg, err := mysql.ParseDSN(rawDSN)
	if err != nil || cfg.Net != "tcp" || cfg.Addr != "127.0.0.1:23309" || !strings.HasPrefix(cfg.DBName, "studio_schema_test") {
		t.Skip("MySQL schema test requires the guarded local disposable database endpoint and studio_schema_test prefix")
	}
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	exists, err := mysqlTableExists(ctx, db, "authorization_predicates")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Skip("guarded MySQL database already has authorization_predicates; use an empty studio_schema_test database")
	}
	defer func() { _, _ = db.ExecContext(context.Background(), "DROP TABLE IF EXISTS authorization_predicates") }()
	ddl, err := schema.MySQLTableStatement("authorization_predicates")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE authorization_predicates DROP INDEX uq_authorization_predicate_type`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE authorization_predicates MODIFY COLUMN package_path VARCHAR(1000) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL, MODIFY COLUMN type_name VARCHAR(300) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO authorization_predicates(name,title,package_path,type_name,owner_id,status,etag,created_at,updated_at) VALUES('nonascii','Non ASCII','pkg.é','pkg.Type','test','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if err := MigrateAuthorizationPredicateKeyMySQL(ctx, db); err == nil {
		t.Fatal("MySQL collation migration accepted a non-ASCII package path")
	}
	packagePath, err := mysqlColumnCollation(ctx, db, "package_path")
	if err != nil || packagePath.characterSet != "utf8mb4" || packagePath.collation != "utf8mb4_bin" {
		t.Fatalf("failed preflight altered package_path collation: %+v err=%v", packagePath, err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM authorization_predicates WHERE name='nonascii'`); err != nil {
		t.Fatal(err)
	}
	if err := MigrateAuthorizationPredicateKeyMySQL(ctx, db); err != nil {
		t.Fatal(err)
	}
	pathX, pathY := strings.Repeat("a", 999)+"x", strings.Repeat("a", 999)+"y"
	for _, row := range []struct{ name, path, typeName string }{
		{"path-x", pathX, "com.example.Café"},
		{"path-y", pathY, "com.example.Café"},
		{"case-lower", pathX, "com.example.café"},
	} {
		if _, err := db.ExecContext(ctx, `INSERT INTO authorization_predicates(name,title,package_path,type_name,owner_id,status,etag,created_at,updated_at) VALUES(?,?,?,?,'test','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, row.name, row.name, row.path, row.typeName); err != nil {
			t.Fatalf("binary key rejected distinct value %q/%q: %v", row.path, row.typeName, err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO authorization_predicates(name,title,package_path,type_name,owner_id,status,etag,created_at,updated_at) VALUES('duplicate','Duplicate',?,'com.example.Café','test','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, pathX); err == nil {
		t.Fatal("exact duplicate full key was not rejected")
	}
}
