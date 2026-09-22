package connectivity

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/viant/datly-studio/internal/connectorsecret"
	"github.com/viant/datly-studio/sdk"
	_ "modernc.org/sqlite"
)

func TestResolveDriverUsesLinkedSQLiteDriver(t *testing.T) {
	if actual := resolveDriver("sqlite"); actual != "sqlite" {
		t.Fatalf("resolveDriver(sqlite) = %q, want sqlite", actual)
	}
}

func TestSQLProbeResolvesServerHeldSecret(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "probe.db")
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TABLE ready(id INTEGER)`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	probe := SQLProbe{Secrets: connectorsecret.ResolverFunc(func(_ context.Context, template, reference string) (string, error) {
		if template != "${Database}" || reference != "secret://database" {
			t.Fatalf("template=%q reference=%q", template, reference)
		}
		return dsn, nil
	})}
	result, err := probe.Probe(context.Background(), &sdk.Connector{Name: "main", Driver: "sqlite", DSNTemplate: "${Database}", SecretRef: "secret://database"})
	if err != nil || result.Status != "passed" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}
