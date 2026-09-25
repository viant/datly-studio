package otheractivepredicate

import (
	"context"
	"strings"
	"testing"
)

type reportID string

func (value reportID) ExcludedPublicationReport() string { return string(value) }

func TestOtherActiveRequiresExcludedReport(t *testing.T) {
	if _, err := (&OtherActive{}).Compute(context.Background(), nil); err == nil {
		t.Fatal("missing binding accepted")
	}
	if _, err := (&OtherActive{Input: reportID("")}).Compute(context.Background(), nil); err == nil {
		t.Fatal("empty excluded ID accepted")
	}
	criteria, err := (&OtherActive{Input: reportID("current")}).Compute(context.Background(), nil)
	if err != nil || len(criteria.Placeholders) != 1 || criteria.Placeholders[0] != "current" || !strings.Contains(criteria.Expression, "publication_status = 'active'") {
		t.Fatalf("criteria=%+v err=%v", criteria, err)
	}
}
