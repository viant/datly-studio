package auditstore

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/viant/datly-studio/schema"
	"os"
	"testing"
)

func TestNativeAuditControlsMySQL(t *testing.T) {
	dsn := os.Getenv("STUDIO_AUDIT_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set STUDIO_AUDIT_MYSQL_DSN to an empty disposable database")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.Initialize(t.Context(), db, "mysql"); err != nil {
		t.Fatal(err)
	}
	checkNativeAuditControls(t, db)
}
