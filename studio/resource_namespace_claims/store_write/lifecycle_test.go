package store_write

import (
	"context"
	"errors"
	"testing"
	"time"

	xhandler "github.com/viant/xdatly/handler"
)

func TestNamespaceClaimRulesPreserveOwnerAndAudit(t *testing.T) {
	now := time.Now().UTC()
	previous := &StoredClaim{Namespace: "owner.docs", ReportId: "report-1",
		CreatedAt: now.Add(-time.Hour), CreatedBy: "alice"}
	state := xhandler.LifecycleContext[StoredClaim, xhandler.NoParent, Output]{
		EntityState: xhandler.EntityState[StoredClaim, xhandler.NoParent]{Previous: previous}}
	row := func(reportID string) *StoredClaim {
		return &StoredClaim{Namespace: "owner.docs", ReportId: reportID,
			CreatedAt: now, CreatedBy: "bob", UpdatedAt: now, UpdatedBy: "bob",
			Has: &StoredClaimHas{Namespace: true, ReportId: true, CreatedAt: true,
				CreatedBy: true, UpdatedAt: true, UpdatedBy: true, ShouldDelete: true}}
	}
	rules := &NamespaceClaimRules{Input: &Input{Operation: "acquire"}}
	valid := row("report-1")
	if err := rules.Init(context.Background(), valid, state); err != nil ||
		!valid.CreatedAt.Equal(previous.CreatedAt) || valid.CreatedBy != "alice" {
		t.Fatalf("same-owner acquisition=%+v err=%v", valid, err)
	}
	var conflict *xhandler.Conflict
	if err := rules.Init(context.Background(), row("report-2"), state); !errors.As(err, &conflict) {
		t.Fatalf("cross-owner acquisition error=%v", err)
	}
	rules.Input.Operation = "release"
	release := &StoredClaim{Namespace: "owner.docs", ReportId: "report-1", ShouldDelete: true,
		Has: &StoredClaimHas{Namespace: true, ReportId: true, ShouldDelete: true}}
	if err := rules.Init(context.Background(), release, state); err != nil {
		t.Fatalf("owner release rejected: %v", err)
	}
	release.ReportId = "report-2"
	if err := rules.Init(context.Background(), release, state); !errors.As(err, &conflict) {
		t.Fatalf("cross-owner release error=%v", err)
	}
}
