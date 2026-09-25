package store_insert

import (
	"context"
	"strings"
	"testing"
	"time"

	xhandler "github.com/viant/xdatly/handler"
)

func TestPublicationInsertRulesRequirePendingExactVersion(t *testing.T) {
	version := 2
	revision := "report:2:7"
	row := &StoredPublication{ReportId: "report", ActiveVersionNo: version, DesiredVersionNo: &version,
		DesiredGeneration: 7, PublicationStatus: "pending", RuntimeRevision: &revision,
		SpecHash: strings.Repeat("a", 64), PublishedBy: "owner", PublishedAt: time.Now().UTC()}
	rules := &PublicationInsertRules{}
	state := xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]{}
	if err := rules.Init(context.Background(), row, state); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*StoredPublication){
		func(value *StoredPublication) { value.PublicationStatus = "active" },
		func(value *StoredPublication) { other := 1; value.DesiredVersionNo = &other },
		func(value *StoredPublication) { active := int64(7); value.ActiveGeneration = &active },
		func(value *StoredPublication) { value.PublishedBy = "" },
		func(value *StoredPublication) { failure := `[]`; value.FailureJson = &failure },
	} {
		invalid := *row
		mutate(&invalid)
		if err := rules.Init(context.Background(), &invalid, state); err == nil {
			t.Fatalf("invalid initial publication accepted: %+v", invalid)
		}
	}
}
