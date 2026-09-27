package componentarchive

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestArchiveBudgets(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
	}{
		{"empty source", CheckSource("  ")},
		{"large source", CheckSource(strings.Repeat("x", MaxSourceBytes+1))},
		{"too many files", CheckBudget(MaxResourceFiles+1, 0)},
		{"too many resource bytes", CheckBudget(1, MaxResourceBytes+1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if !errors.Is(test.err, ErrInvalid) {
				t.Fatalf("error = %v, want ErrInvalid", test.err)
			}
		})
	}
	if err := CheckBudget(MaxResourceFiles, MaxResourceBytes); err != nil {
		t.Fatalf("at limit: %v", err)
	}
}

func TestBuildRejectsOversizedContent(t *testing.T) {
	_, err := Build(context.Background(), "report", 1, "SELECT 1", []File{{
		ReportID: "report", VersionNo: 1, ResourcePath: "data.txt", Content: make([]byte, MaxResourceBytes+1),
	}})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("error = %v, want ErrInvalid", err)
	}
}
