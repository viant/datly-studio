package store_insert

import (
	"context"
	"testing"
	"time"

	xhandler "github.com/viant/xdatly/handler"
)

func TestGenerationInsertRulesRequireBuildingState(t *testing.T) {
	row := &StoredGeneration{GenerationNo: 3, SourceRevision: "report:1:3",
		Status: "building", BuildManifestJson: []byte(`{}`), RequestedBy: "owner", RequestedAt: time.Now().UTC()}
	rules := &GenerationInsertRules{}
	state := xhandler.LifecycleContext[StoredGeneration, xhandler.NoParent, Output]{}
	if err := rules.Init(context.Background(), row, state); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*StoredGeneration){
		func(value *StoredGeneration) { value.GenerationNo = 0 },
		func(value *StoredGeneration) { value.Status = "active" },
		func(value *StoredGeneration) { value.ReportCount = 1 },
		func(value *StoredGeneration) { value.BuildManifestJson = []byte(`bad`) },
		func(value *StoredGeneration) { value.RequestedBy = "" },
	} {
		invalid := *row
		mutate(&invalid)
		if err := rules.Init(context.Background(), &invalid, state); err == nil {
			t.Fatalf("invalid generation accepted: %+v", invalid)
		}
	}
}
