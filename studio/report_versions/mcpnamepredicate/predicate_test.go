package mcpnamepredicate

import (
	"context"
	"strings"
	"testing"
)

type reportID string

func (value reportID) MCPNameReportID() string { return string(value) }

func TestOtherReportCandidateRequiresExactIdentity(t *testing.T) {
	if _, err := (&OtherReportCandidate{}).Compute(context.Background(), nil); err == nil {
		t.Fatal("missing bound input accepted")
	}
	if _, err := (&OtherReportCandidate{Input: reportID("")}).Compute(context.Background(), nil); err == nil {
		t.Fatal("empty report identity accepted")
	}
	criteria, err := (&OtherReportCandidate{Input: reportID("r-1")}).Compute(context.Background(), nil)
	if err != nil || len(criteria.Placeholders) != 1 || criteria.Placeholders[0] != "r-1" ||
		!strings.Contains(criteria.Expression, "MAX(v2.version_no)") || !strings.Contains(criteria.Expression, "active_version_no") {
		t.Fatalf("criteria=%+v err=%v", criteria, err)
	}
}
