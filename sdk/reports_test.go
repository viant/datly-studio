package sdk

import "testing"

func TestOwnerPackageSegmentUsesJWTSubject(t *testing.T) {
	if got := OwnerPackageSegment("awitas"); got != "awitas" {
		t.Fatalf("simple owner=%q", got)
	}
	if got := OwnerPackageSegment("Adrian.Witas_test+studio@viantinc.com"); got != "adrianwitasteststudio" {
		t.Fatalf("email-shaped JWT subject=%q", got)
	}
	if OwnerPackageSegment("Adrian.Witas@viantinc.com") != "adrianwitas" {
		t.Fatal("owner package segment is not deterministic")
	}
}
