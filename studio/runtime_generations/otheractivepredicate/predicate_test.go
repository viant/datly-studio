package otheractivepredicate

import (
	"context"
	"strings"
	"testing"
)

type generation int64

func (value generation) ExcludedRuntimeGeneration() int64 { return int64(value) }

func TestOtherActiveRequiresExcludedGeneration(t *testing.T) {
	if _, err := (&OtherActive{}).Compute(context.Background(), nil); err == nil {
		t.Fatal("missing binding accepted")
	}
	if _, err := (&OtherActive{Input: generation(0)}).Compute(context.Background(), nil); err == nil {
		t.Fatal("zero generation accepted")
	}
	criteria, err := (&OtherActive{Input: generation(7)}).Compute(context.Background(), nil)
	if err != nil || len(criteria.Placeholders) != 1 || criteria.Placeholders[0] != int64(7) || !strings.Contains(criteria.Expression, "status = 'active'") {
		t.Fatalf("criteria=%+v err=%v", criteria, err)
	}
}
