package store_stage

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	xhandler "github.com/viant/xdatly/handler"
)

func TestPublicationStageRulesMatchPreviousGeneration(t *testing.T) {
	previousGeneration := int64(4)
	previous := &StoredPublication{ReportId: "report", DesiredGeneration: &previousGeneration, PublicationStatus: "active"}
	state := xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]{
		EntityState: xhandler.EntityState[StoredPublication, xhandler.NoParent]{Previous: previous}}
	version := 2
	revision := "report:2:5"
	now := time.Now().UTC()
	row := func() *StoredPublication {
		expected := previousGeneration
		return &StoredPublication{ReportId: "report", DesiredVersionNo: &version, DesiredGeneration: &expected,
			PublicationStatus: "pending", RuntimeRevision: &revision, SpecHash: strings.Repeat("a", 64),
			PublishedBy: "owner", PublishedAt: &now,
			Has: &StoredPublicationHas{ReportId: true, DesiredVersionNo: true, DesiredGeneration: true,
				PublicationStatus: true, RuntimeRevision: true, SpecHash: true,
				PublishedBy: true, PublishedAt: true, FailureJson: true}}
	}
	rules := &PublicationStageRules{Input: &Input{NextGeneration: 5}}
	valid := row()
	if err := rules.Init(context.Background(), valid, state); err != nil || *valid.DesiredGeneration != 5 {
		t.Fatalf("restaged publication=%+v err=%v", valid, err)
	}
	legacy := row()
	legacy.SpecHash = "legacy-hash"
	if err := rules.Init(context.Background(), legacy, state); err != nil {
		t.Fatalf("authoritative legacy spec hash was rejected: %v", err)
	}
	stale := row()
	*stale.DesiredGeneration--
	var conflict *xhandler.Conflict
	if err := rules.Init(context.Background(), stale, state); !errors.As(err, &conflict) {
		t.Fatalf("stale desired generation error=%v", err)
	}
	previous.PublicationStatus = "pending"
	if err := rules.Init(context.Background(), row(), state); !errors.As(err, &conflict) {
		t.Fatalf("already staged publication error=%v", err)
	}
	previous.PublicationStatus = "active"
	rules.Input.NextGeneration = 4
	if err := rules.Init(context.Background(), row(), state); err == nil {
		t.Fatal("non-advancing generation accepted")
	}
}
