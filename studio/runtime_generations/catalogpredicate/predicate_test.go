package catalogpredicate

import (
	"context"
	"strings"
	"testing"
)

type scope struct {
	generation int64
	subject    string
	scoped     bool
}

func (value scope) RuntimeReaderScope() (int64, string, bool) {
	return value.generation, value.subject, value.scoped
}

func TestReaderScopeRequiresGenerationAndScopedSubject(t *testing.T) {
	if _, err := (&ReaderScope{}).Compute(context.Background(), nil); err == nil {
		t.Fatal("missing binding accepted")
	}
	if _, err := (&ReaderScope{Input: scope{scoped: true}}).Compute(context.Background(), nil); err == nil {
		t.Fatal("missing generation accepted")
	}
	if _, err := (&ReaderScope{Input: scope{generation: 4, scoped: true}}).Compute(context.Background(), nil); err == nil {
		t.Fatal("missing subject accepted")
	}
	unscoped, err := (&ReaderScope{Input: scope{generation: 4}}).Compute(context.Background(), nil)
	if err != nil || len(unscoped.Placeholders) != 1 || strings.Contains(unscoped.Expression, "report_acl") {
		t.Fatalf("unscoped=%+v err=%v", unscoped, err)
	}
	scoped, err := (&ReaderScope{Input: scope{generation: 4, subject: "owner", scoped: true}}).Compute(context.Background(), nil)
	if err != nil || len(scoped.Placeholders) != 3 || !strings.Contains(scoped.Expression, "can_publish") {
		t.Fatalf("scoped=%+v err=%v", scoped, err)
	}
}
