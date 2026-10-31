package bffauth

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/viant/datly-studio/schema"
	"os"
	"testing"
)

func TestNativeRefreshLeaseMySQL(t *testing.T) {
	dsn := os.Getenv("STUDIO_SESSION_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set STUDIO_SESSION_MYSQL_DSN to an empty disposable database")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(4)
	if err = schema.Initialize(t.Context(), db, "mysql"); err != nil {
		t.Fatal(err)
	}
	checkDatlyRefreshLease(t, db)
	checkDatlyLeaseContenders(t, db)
}
