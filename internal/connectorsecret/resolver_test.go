package connectorsecret

import (
	"context"
	"github.com/viant/datly/constant"
	"os"
	"path/filepath"
	"strings"
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

func TestDeploymentConstantsPreserveSecretExpansionAndTemplate(t *testing.T) {
	values, err := constant.New(map[string]string{"Project": "ar-e2e", "Dataset": "studies"})
	if err != nil {
		t.Fatal(err)
	}
	const template = "bigquery://${Project}/${Dataset}?credential=${Secret}"
	resolver := ResolverFunc(func(_ context.Context, expanded, reference string) (string, error) {
		if reference != "owned-secret" || expanded != "bigquery://ar-e2e/studies?credential=${Secret}" {
			t.Fatalf("unexpected resolution boundary")
		}
		return strings.ReplaceAll(expanded, "${Secret}", "test-only"), nil
	})
	for _, ctx := range []context.Context{context.Background(), WithConstants(context.Background(), values)} {
		result, err := Resolve(ctx, resolver, template, "owned-secret", values)
		if err != nil || result != "bigquery://ar-e2e/studies?credential=test-only" {
			t.Fatalf("constant/secret resolution failed: %v", err)
		}
	}
	if template != "bigquery://${Project}/${Dataset}?credential=${Secret}" {
		t.Fatal("source template mutated")
	}
	result, err := Resolve(WithConstants(context.Background(), values), nil, "bigquery://${Project}/${Dataset}", "")
	if err != nil || result != "bigquery://ar-e2e/studies" {
		t.Fatalf("context constants not applied: %v", err)
	}
}
