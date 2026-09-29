package catalogpredicate

import (
	"context"
	"testing"
)

type scopeInput struct {
	subject   string
	scoped    bool
	namespace *string
}

func (input scopeInput) ReportCatalogScope() (string, bool) { return input.subject, input.scoped }
func (input scopeInput) ReportCatalogNamespace() *string    { return input.namespace }

func TestTypedNamespaceSelectionCannotFallBackToUnscopedRead(t *testing.T) {
	invalid := "invalid"
	input := scopeInput{subject: "viewer", scoped: true, namespace: &invalid}
	if _, err := (&ReportCatalogRead{Input: input}).Compute(context.Background(), nil); err == nil {
		t.Fatal("typed invalid selection fell back to an unscoped read")
	}
	other := "other"
	if _, err := (&ReportCatalogRead{Input: input, NamespaceID: &other}).Compute(context.Background(), nil); err == nil {
		t.Fatal("conflicting typed and header selections accepted")
	}
}

func TestReportCatalogReadRequiresScopedSubject(t *testing.T) {
	ctx := context.Background()
	if _, err := (&ReportCatalogRead{}).Compute(ctx, nil); err == nil {
		t.Fatal("missing bound scope was accepted")
	}
	if _, err := (&ReportCatalogRead{Input: scopeInput{scoped: true}}).Compute(ctx, nil); err == nil {
		t.Fatal("empty scoped principal was accepted")
	}
	trusted, err := (&ReportCatalogRead{Input: scopeInput{}}).Compute(ctx, nil)
	if err != nil || trusted.Expression != "1=1" {
		t.Fatalf("trusted scope=%+v err=%v", trusted, err)
	}
	owned, err := (&ReportCatalogRead{Input: scopeInput{subject: "viewer", scoped: true}}).Compute(ctx, nil)
	if err != nil || len(owned.Placeholders) != 2 || owned.Placeholders[0] != "viewer" || owned.Placeholders[1] != "viewer" {
		t.Fatalf("scoped criteria=%+v err=%v", owned, err)
	}
}
