package connectivity

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/viant/datly-studio/internal/connectorsecret"
	"github.com/viant/datly-studio/sdk"
	// Match the native executable, where a transitive import registers the CGO stub too.
	_ "github.com/mattn/go-sqlite3"
	"modernc.org/sqlite"
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

func init() { sql.Register("linked_test_sqlite", &sqlite.Driver{}) }

func TestSQLProbeAcceptsHostLinkedDriver(t *testing.T) {
	result, err := (SQLProbe{}).Probe(context.Background(), &sdk.Connector{
		Name: "custom", Driver: "linked_test_sqlite", DSNTemplate: "file:" + filepath.Join(t.TempDir(), "custom.db"),
	})
	if err != nil || result == nil || result.Status != "passed" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestSQLProbeRejectsUnlinkedDriver(t *testing.T) {
	result, err := (SQLProbe{}).Probe(context.Background(), &sdk.Connector{Name: "missing", Driver: "not_linked_driver", DSNTemplate: "unused"})
	if err == nil || result != nil {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestSQLProbeMissingLocalSecretHasActionableDiagnostic(t *testing.T) {
	_, err := (SQLProbe{}).Probe(context.Background(), &sdk.Connector{Name: "missing-secret", Driver: "sqlite", DSNTemplate: "file:unused", SecretRef: filepath.Join(t.TempDir(), "missing.enc") + "|blowfish://default"})
	code, message := Diagnostic(err)
	if code != "secret_not_found" {
		t.Fatalf("diagnostic code=%s message=%s", code, message)
	}
}
