package schema

import (
	"context"
	"strings"
	"testing"
)

func TestInitializeCreatesCanonicalStorageAndReopensReadOnly(t *testing.T) {
	db := openSQLite(t, "canonical_initialize")
	defer db.Close()
	ctx := context.Background()
	if err := Initialize(ctx, db, "sqlite"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO connectors(name,driver,owner_id,status,created_at,updated_at) VALUES('synthetic','sqlite','owner','draft',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("PRAGMA query_only=ON"); err != nil {
		t.Fatal(err)
	}
	if err := Initialize(ctx, db, "sqlite"); err != nil {
		t.Fatalf("existing canonical storage must require no writes: %v", err)
	}
	assertCount(t, db, "SELECT COUNT(*) FROM connectors WHERE name='synthetic'", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM schema_version", 1)
	assertTablePresence(t, ctx, db, "sqlx_sequence_reservations", false)
}

func TestInitializeRejectsIncompleteStorageWithoutRepair(t *testing.T) {
	db := openSQLite(t, "canonical_incomplete")
	defer db.Close()
	ctx := context.Background()
	if err := Initialize(ctx, db, "sqlite"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("DROP TABLE bff_sessions"); err != nil {
		t.Fatal(err)
	}
	if err := Initialize(ctx, db, "sqlite"); err == nil {
		t.Fatal("incomplete storage accepted")
	}
	assertTablePresence(t, ctx, db, "bff_sessions", false)
}

func TestInitializeRejectsOldShapeWithoutAlteringIt(t *testing.T) {
	db := openSQLite(t, "canonical_old_shape")
	defer db.Close()
	ctx := context.Background()
	if _, err := db.Exec("CREATE TABLE connectors(name TEXT PRIMARY KEY);INSERT INTO connectors VALUES('preserved')"); err != nil {
		t.Fatal(err)
	}
	if err := Initialize(ctx, db, "sqlite"); err == nil {
		t.Fatal("old shape was accepted")
	}
	assertCount(t, db, "SELECT COUNT(*) FROM connectors WHERE name='preserved'", 1)
	assertTablePresence(t, ctx, db, "schema_version", false)
	assertTablePresence(t, ctx, db, "namespaces", false)
}

func TestInitializeRejectsGenerationCounterDowngradeWithoutRepair(t *testing.T) {
	db := openSQLite(t, "generation_identity_required")
	defer db.Close()
	ctx := context.Background()
	ddl, err := canonicalDDL("sqlite")
	if err != nil {
		t.Fatal(err)
	}
	ddl = strings.Replace(ddl, "INTEGER PRIMARY KEY AUTOINCREMENT", "INTEGER PRIMARY KEY", 1)
	if _, err = db.ExecContext(ctx, ddl); err != nil {
		t.Fatal(err)
	}
	if err = Initialize(ctx, db, "sqlite"); err == nil || !strings.Contains(err.Error(), "AUTOINCREMENT") {
		t.Fatalf("non-native identity accepted: %v", err)
	}
	assertTablePresence(t, ctx, db, "sqlx_sequence_reservations", false)
}
