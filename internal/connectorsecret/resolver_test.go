package connectorsecret

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveUsesInjectedServerResolver(t *testing.T) {
	var gotTemplate, gotReference string
	resolved, err := Resolve(context.Background(), ResolverFunc(func(_ context.Context, template, reference string) (string, error) {
		gotTemplate, gotReference = template, reference
		return "resolved-dsn", nil
	}), "user=${Username}", "secret://database")
	if err != nil || resolved != "resolved-dsn" || gotTemplate != "user=${Username}" || gotReference != "secret://database" {
		t.Fatalf("resolved=%q template=%q reference=%q err=%v", resolved, gotTemplate, gotReference, err)
	}
}

func TestResolveDefaultsToSCYForServerHeldSecretReference(t *testing.T) {
	location := filepath.Join(t.TempDir(), "connector-secret.txt")
	const dsn = "file:scy-probe.db?mode=ro"
	if err := os.WriteFile(location, []byte(dsn), 0600); err != nil {
		t.Fatal(err)
	}
	resolved, err := Resolve(context.Background(), nil, "", location)
	if err != nil || resolved != dsn {
		t.Fatalf("default SCY resolver produced expected DSN=%t err=%v", resolved == dsn, err)
	}
}
