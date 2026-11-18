package sqltransport

import (
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/authoring/readerbuilder"
	"github.com/viant/datly/spec"
)

const warmupContributorSource = `#setting($_ = $connector('source'))
#setting($_ = $route('/records','GET'))
#setting($_ = $cache_warmup_cases('recent','Period=PRIVATE_CASE_CANARY'))
#setting($_ = $cache_warmup('ID','Period=legacy'))
#define($_ = $Period<string>(query/period).Optional())
SELECT r.*, c.*, use_cache(r,'root-storage'), use_cache(c,'child-storage'), cache_warmup(r,'','Limit=0'),
cache_warmup(c,'parent_id','MaxCases=4','Limit=0','CaseRefs=recent','Period=older'),
cache_warmup(c,'parent_id','MaxCases=4','Limit=0','Period=oldest'), cache_warmup(c,'ID','Name=by_id')
FROM records r LEFT JOIN children c ON c.parent_id=r.ID`

type warmupContributorProjection struct {
	Component       *spec.Component                    `json:"component"`
	TargetedWarmups []readerbuilder.TargetedWarmupPlan `json:"targetedWarmups"`
}

func TestWarmupContributorsInspectionAndAtomicBuilder(t *testing.T) {
	f := newImportFixture(t)
	version, err := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: warmupContributorSource})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := f.client.Versions().Inspect(f.ctx, f.report.ID, version.VersionNo)
	if err != nil {
		t.Fatal(err)
	}
	var structure warmupContributorProjection
	if err = json.Unmarshal(snapshot.Structure, &structure); err != nil {
		t.Fatal(err)
	}
	if len(structure.TargetedWarmups) != 3 || structure.TargetedWarmups[1].View != "c" || len(structure.TargetedWarmups[1].Contributors) != 2 {
		t.Fatalf("missing native contributors: %s", snapshot.Structure)
	}
	for _, plan := range structure.TargetedWarmups {
		if !plan.Editable || plan.Diagnostic != "" || len(plan.Contributors) == 0 {
			t.Fatalf("editable metadata changed: %+v", plan)
		}
	}
	if structure.TargetedWarmups[0].Settings.MaxCases != nil || structure.TargetedWarmups[0].Settings.Limit == nil || *structure.TargetedWarmups[0].Settings.Limit != 0 || structure.TargetedWarmups[2].Settings.MaxCases != nil || structure.TargetedWarmups[2].Settings.Limit != nil {
		t.Fatal("omission and zero were conflated")
	}
	parent := structure.TargetedWarmups[1].Settings
	if parent.MaxCases == nil || *parent.MaxCases != 4 || !reflect.DeepEqual(parent.CaseRefs, []string{"recent"}) || len(parent.Cases) != 2 {
		t.Fatalf("authored groups were expanded or lost: %+v", parent)
	}
	calls := structure.TargetedWarmups[1].Contributors
	operations := []readerbuilder.Operation{}
	for _, call := range calls {
		args := append([]string(nil), call.Args...)
		for i, arg := range args {
			if arg == "'Limit=0'" {
				args[i] = "'Limit=25'"
			}
		}
		operations = append(operations, readerbuilder.Operation{Type: readerbuilder.OperationUpdateFunction, Function: &readerbuilder.FunctionMutation{Name: call.Name, Occurrence: call.Occurrence, ExpectedArgs: call.Args, Args: args}})
	}
	apply := func(revision int64, ops []readerbuilder.Operation) (*sdk.ReaderBuilderResult, error) {
		t.Helper()
		raw, err := json.Marshal(readerbuilder.Operation{Type: readerbuilder.OperationBatch, Operations: ops})
		if err != nil {
			t.Fatal(err)
		}
		return f.client.Versions().ApplyReaderCommand(f.ctx, f.report.ID, version.VersionNo, sdk.ReaderBuilderCommand{ExpectedSourceRevision: revision, Operation: raw})
	}
	assertUnchanged := func() {
		t.Helper()
		current, err := f.client.Versions().Get(f.ctx, f.report.ID, version.VersionNo)
		if err != nil || current.SourceRevision != version.SourceRevision || current.AuthoredDQL != warmupContributorSource || current.GeneratedDQL != version.GeneratedDQL {
			t.Fatalf("rejected batch changed source: %+v err=%v", current, err)
		}
	}
	_, err = apply(version.SourceRevision+1, operations)
	var conflict *sdk.Error
	if !errors.As(err, &conflict) || conflict.Code != sdk.ErrorConflict || conflict.Field != "sourceRevision" {
		t.Fatalf("stale revision accepted: %v", err)
	}
	assertUnchanged()
	invalid := append([]readerbuilder.Operation(nil), operations...)
	bad := *invalid[1].Function
	bad.ExpectedArgs = append([]string(nil), bad.ExpectedArgs...)
	bad.ExpectedArgs[0] = "other"
	invalid[1].Function = &bad
	rejected, err := apply(version.SourceRevision, invalid)
	if err != nil || rejected.Applied {
		t.Fatalf("mismatch accepted: %+v err=%v", rejected, err)
	}
	assertUnchanged()
	updated, err := apply(version.SourceRevision, operations)
	if err != nil || !updated.Applied || updated.Inspection.Version.SourceRevision != version.SourceRevision+1 {
		t.Fatalf("update failed: %+v err=%v", updated, err)
	}
	if !strings.Contains(updated.Inspection.DQL, "CaseRefs=recent") || !strings.Contains(updated.Inspection.DQL, "Period=older") || !strings.Contains(updated.Inspection.DQL, "Period=oldest") || !strings.Contains(updated.Inspection.DQL, "PRIVATE_CASE_CANARY") {
		t.Fatal("lossy contributor update")
	}
	var after warmupContributorProjection
	if err = json.Unmarshal(updated.Inspection.Structure, &after); err != nil {
		t.Fatal(err)
	}
	for i, before := range structure.TargetedWarmups {
		want := before.Settings.Clone()
		if i == 1 {
			limit := 25
			want.Limit = &limit
		}
		if !reflect.DeepEqual(want, after.TargetedWarmups[i].Settings) {
			t.Fatalf("lossy plan %d update: got=%+v want=%+v", i, after.TargetedWarmups[i].Settings, want)
		}
	}
	rootBindings := structure.Component.RootView.Source.Bindings.Clone()
	childBindings := structure.Component.RootView.Relations[0].View.Source.Bindings.Clone()
	limit := 25
	childBindings.InlineWarmups[0].Limit = &limit
	if !reflect.DeepEqual(rootBindings, after.Component.RootView.Source.Bindings) || !reflect.DeepEqual(childBindings, after.Component.RootView.Relations[0].View.Source.Bindings) || !reflect.DeepEqual(structure.Component.Settings, after.Component.Settings) {
		t.Fatal("storage bindings, shared cases, or component plan changed")
	}
	readback, err := f.client.Versions().Inspect(f.ctx, f.report.ID, version.VersionNo)
	if err != nil {
		t.Fatal(err)
	}
	var saved, returned warmupContributorProjection
	if err = json.Unmarshal(readback.Structure, &saved); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(updated.Inspection.Structure, &returned); err != nil || !reflect.DeepEqual(saved, returned) {
		t.Fatalf("saved contributor identities did not round-trip: %v", err)
	}
	calls = append([]readerbuilder.FunctionOccurrence(nil), saved.TargetedWarmups[1].Contributors...)
	sort.Slice(calls, func(i, j int) bool { return calls[i].Occurrence > calls[j].Occurrence })
	operations = nil
	for _, call := range calls {
		operations = append(operations, readerbuilder.Operation{Type: readerbuilder.OperationRemoveFunction, Function: &readerbuilder.FunctionMutation{Name: call.Name, Occurrence: call.Occurrence, ExpectedArgs: call.Args}})
	}
	removed, err := apply(readback.Version.SourceRevision, operations)
	if err != nil || !removed.Applied || strings.Contains(removed.Inspection.DQL, "parent_id',") || !strings.Contains(removed.Inspection.DQL, "$cache_warmup_cases(") {
		t.Fatalf("contributor removal failed or removed shared cases: %+v err=%v", removed, err)
	}
	var remaining warmupContributorProjection
	if err = json.Unmarshal(removed.Inspection.Structure, &remaining); err != nil {
		t.Fatal(err)
	}
	childBindings.InlineWarmups = childBindings.InlineWarmups[1:]
	if len(remaining.TargetedWarmups) != 2 || !reflect.DeepEqual(after.TargetedWarmups[0].Settings, remaining.TargetedWarmups[0].Settings) || !reflect.DeepEqual(after.TargetedWarmups[2].Settings, remaining.TargetedWarmups[1].Settings) || !reflect.DeepEqual(rootBindings, remaining.Component.RootView.Source.Bindings) || !reflect.DeepEqual(childBindings, remaining.Component.RootView.Relations[0].View.Source.Bindings) || !reflect.DeepEqual(structure.Component.Settings, remaining.Component.Settings) {
		t.Fatal("removal changed unrelated plans, bindings, or shared cases")
	}
	if _, err = f.db.ExecContext(f.ctx, "INSERT INTO component_acl(report_id,subject_type,subject_id,can_view,can_run,can_publish,can_use_dql) VALUES(?,'user','runner',1,1,1,0)", f.report.ID); err != nil {
		t.Fatal(err)
	}
	runner := sdk.WithPrincipal(f.ctx, sdk.Principal{Subject: "runner"})
	redacted, err := f.client.Versions().Inspect(runner, f.report.ID, version.VersionNo)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(redacted)
	if redacted.Capabilities.CanUseDQL || strings.Contains(string(raw), "contributors") || strings.Contains(string(raw), "targetedWarmups") || strings.Contains(string(raw), "PRIVATE_CASE_CANARY") {
		t.Fatalf("source-only contributor metadata leaked: %s", raw)
	}
}

func TestWarmupContributorReadOnlyInspection(t *testing.T) {
	f := newImportFixture(t)
	source := strings.Replace(warmupContributorSource, "cache_warmup(r,'','Limit=0')", "cache_warmup(r,'','Limit=0') AS warmup", 1)
	version, err := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: source})
	if err != nil {
		t.Fatal(err)
	}
	inspected, err := f.client.Versions().Inspect(f.ctx, f.report.ID, version.VersionNo)
	if err != nil {
		t.Fatal(err)
	}
	var structure warmupContributorProjection
	if err = json.Unmarshal(inspected.Structure, &structure); err != nil {
		t.Fatal(err)
	}
	if len(structure.TargetedWarmups) != 3 {
		t.Fatalf("plans missing: %s", inspected.Structure)
	}
	plan := structure.TargetedWarmups[0]
	if plan.Editable || plan.Diagnostic == "" || plan.Contributors == nil || len(plan.Contributors) != 0 || plan.Settings.Limit == nil || *plan.Settings.Limit != 0 {
		t.Fatalf("read-only native metadata changed: %+v", plan)
	}
	var wire struct {
		TargetedWarmups []map[string]json.RawMessage `json:"targetedWarmups"`
	}
	if err = json.Unmarshal(inspected.Structure, &wire); err != nil {
		t.Fatal(err)
	}
	if string(wire.TargetedWarmups[0]["editable"]) != "false" || string(wire.TargetedWarmups[0]["contributors"]) != "[]" {
		t.Fatalf("read-only wire shape changed: %s", inspected.Structure)
	}
	if !structure.TargetedWarmups[1].Editable || !structure.TargetedWarmups[2].Editable {
		t.Fatal("read-only root changed unrelated child editability")
	}
	if _, err = f.db.ExecContext(f.ctx, "INSERT INTO component_acl(report_id,subject_type,subject_id,can_view,can_run,can_publish,can_use_dql) VALUES(?,'user','runner',1,1,1,0)", f.report.ID); err != nil {
		t.Fatal(err)
	}
	hidden, err := f.client.Versions().Inspect(sdk.WithPrincipal(f.ctx, sdk.Principal{Subject: "runner"}), f.report.ID, version.VersionNo)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(hidden)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"targetedWarmups", "contributors", "PRIVATE_CASE_CANARY", plan.Diagnostic} {
		if strings.Contains(string(raw), secret) {
			t.Fatalf("read-only source metadata leaked: %s", raw)
		}
	}
}
