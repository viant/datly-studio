package store_insert

import (
	"context"
	"strings"
	"testing"
	"time"

	xhandler "github.com/viant/xdatly/handler"
)

func TestVersionInsertRulesRequireInitialRevisionContract(t *testing.T) {
	row := &StoredVersion{ReportId: "report", VersionNo: 1, State: "draft", AuthoringMode: "dql",
		ComponentSpecJson: []byte(`{}`), TypeManifestJson: []byte(`{}`),
		SpecFormatVersion: "studio.v1", SpecHash: strings.Repeat("a", 64),
		CompileStatus: "pending", DatlyVersion: "v1", CompilerVersion: "studio.v1",
		SourceRevision: 1, CreatedBy: "owner", CreatedAt: time.Now().UTC()}
	rules := &VersionInsertRules{}
	state := xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]{}
	if err := rules.Init(context.Background(), row, state); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*StoredVersion){
		func(value *StoredVersion) { value.State = "published" },
		func(value *StoredVersion) { value.AuthoringMode = "unknown" },
		func(value *StoredVersion) { value.SourceRevision = 2 },
		func(value *StoredVersion) { value.ComponentSpecJson = []byte(`invalid`) },
		func(value *StoredVersion) { value.SpecHash = "not-a-hash" },
	} {
		invalid := *row
		mutate(&invalid)
		if err := rules.Init(context.Background(), &invalid, state); err == nil {
			t.Fatalf("invalid initial version accepted: %+v", invalid)
		}
	}
}
