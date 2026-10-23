package schema

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestReleaseMigration25IsExplicitAndRestoresCanonicalSchema(t *testing.T) {
	ctx := context.Background()
	db := openSQLite(t, "artifact_release_migration")
	defer db.Close()
	if err := ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"artifact_release_operations", "artifact_release_channels", "artifact_release_bundles"} {
		if _, err := db.ExecContext(ctx, "DROP TABLE "+table); err != nil {
			t.Fatal(err)
		}
	}
	if err := SetSQLiteVersion(ctx, db, 24); err != nil {
		t.Fatal(err)
	}
	if err := Verify(ctx, db, "sqlite"); err == nil {
		t.Fatal("normal startup silently accepted a store missing release tables")
	}
	upgrade, err := os.ReadFile("migrations/025-artifact-releases.sqlite.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, string(upgrade)); err != nil {
		t.Fatal(err)
	}
	if err := Verify(ctx, db, "sqlite"); err != nil {
		t.Fatal(err)
	}
	version, err := SQLiteVersion(ctx, db)
	if err != nil || version != CanonicalVersion {
		t.Fatalf("schema version=%d err=%v", version, err)
	}
}

func TestReleaseChannelNamesAreBinaryInMySQL(t *testing.T) {
	ddl, err := MySQLDDL()
	if err != nil {
		t.Fatal(err)
	}
	upgrade, err := os.ReadFile("migrations/025-artifact-releases.mysql.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{ddl, string(upgrade)} {
		for _, field := range []string{"name VARCHAR(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin", "channel_name VARCHAR(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin"} {
			if !strings.Contains(strings.Join(strings.Fields(source), " "), field) {
				t.Fatalf("release channel key is case-folded in MySQL: %s", field)
			}
		}
	}
}
