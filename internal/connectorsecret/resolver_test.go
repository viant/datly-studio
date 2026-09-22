package connectorsecret

import (
	"context"
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
