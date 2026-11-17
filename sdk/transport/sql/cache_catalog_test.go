package sqltransport

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/authoring/readerbuilder"
	"github.com/viant/datly/spec"
)

const cacheBindingSource = `#setting($_ = $connector('source'))
#setting($_ = $route('/records','GET'))
#setting($_ = $cache('component','1m').WithLocation('source-cache'))
#setting($_ = $cache_warmup_cases('recent','Period=recent'))
#setting($_ = $cache_warmup('ID','Limit=0'))
#define($_ = $Period<string>(query/period).Optional())
SELECT r.*, c.*, s.*, use_cache(r,'old-root'),use_cache(c,'old-child'),use_cache(s,'unchanged'),
use_connector(c,'source'),cache_warmup(c,'ID','CaseRefs=recent','Limit=0'),cache_warmup(s,'host-policy')
FROM records r LEFT JOIN children c ON c.parent_id=r.ID LEFT JOIN summary s ON s.ID=r.ID`

type cacheBindingStructure struct {
	AvailableCaches []string                           `json:"availableCaches"`
	Component       *spec.Component                    `json:"component"`
	Functions       []readerbuilder.FunctionOccurrence `json:"functions"`
}

func TestHostCacheCatalogAndGuardedBindings(t *testing.T) {
	for _, namedComponent := range []bool{false, true} {
		t.Run(map[bool]string{false: "inline-component", true: "named-component"}[namedComponent], func(t *testing.T) {
			f := newImportFixture(t)
			f.transport.ReaderCaches = map[string]*spec.CacheSettings{
				"shared":      {Name: "not-the-map-key", Enabled: true, Provider: "aerospike://unreachable.invalid:3000/private", Location: "PRIVATE_STORAGE_CANARY", TTL: "1m"},
				"combined":    {Enabled: true, Location: "PRIVATE_COMBINED_CANARY", TTL: "1m", Warmup: &spec.CacheWarmupSettings{IndexColumn: "ID"}},
				"policy-only": {Warmup: &spec.CacheWarmupSettings{IndexColumn: "ID"}},
				"disabled":    {Enabled: false, Location: "PRIVATE_DISABLED_CANARY", TTL: "1m"}, "nil-entry": nil,
			}
			source := cacheBindingSource
			if namedComponent {
				source = strings.Replace(source, "$cache('component','1m').WithLocation('source-cache')", "$cache('shared')", 1)
			}
			version, err := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: source})
			if err != nil {
				t.Fatal(err)
			}
			inspect := func() *sdk.ReaderInspection {
				t.Helper()
				got, err := f.client.Versions().Inspect(f.ctx, f.report.ID, version.VersionNo)
				if err != nil {
					t.Fatal(err)
				}
				return got
			}
			decode := func(got *sdk.ReaderInspection) cacheBindingStructure {
				t.Helper()
				var structure cacheBindingStructure
				if err := json.Unmarshal(got.Structure, &structure); err != nil {
					t.Fatal(err)
				}
				return structure
			}
			snapshot := inspect()
			before := decode(snapshot)
			if !reflect.DeepEqual(before.AvailableCaches, []string{"combined", "shared"}) {
				t.Fatalf("storage catalog=%v", before.AvailableCaches)
			}
			raw, _ := json.Marshal(snapshot)
			for _, secret := range []string{"PRIVATE_", "unreachable.invalid", "not-the-map-key"} {
				if strings.Contains(string(raw), secret) {
					t.Fatalf("host settings leaked: %s", raw)
				}
			}
			var calls []readerbuilder.FunctionOccurrence
			for _, call := range before.Functions {
				if call.Name == "use_cache" && call.Occurrence < 2 {
					calls = append(calls, call)
				}
			}
			if len(calls) != 2 {
				t.Fatal("fixture cache calls missing")
			}
			operation := func(call readerbuilder.FunctionOccurrence, name string) readerbuilder.Operation {
				return readerbuilder.Operation{Type: readerbuilder.OperationUpdateFunction, Function: &readerbuilder.FunctionMutation{Name: call.Name, Occurrence: call.Occurrence, ExpectedArgs: call.Args, Args: []string{call.Args[0], "'" + name + "'"}}}
			}
			apply := func(revision int64, ops ...readerbuilder.Operation) (*sdk.ReaderBuilderResult, error) {
				t.Helper()
				raw, err := json.Marshal(readerbuilder.Operation{Type: readerbuilder.OperationBatch, Operations: ops})
				if err != nil {
					t.Fatal(err)
				}
				return f.client.Versions().ApplyReaderCommand(f.ctx, f.report.ID, version.VersionNo, sdk.ReaderBuilderCommand{ExpectedSourceRevision: revision, Operation: raw})
			}
			unchanged := func() {
				t.Helper()
				current, err := f.client.Versions().Get(f.ctx, f.report.ID, version.VersionNo)
				if err != nil || current.SourceRevision != version.SourceRevision || current.AuthoredDQL != source || current.GeneratedDQL != version.GeneratedDQL {
					t.Fatalf("rejected batch changed source: %+v err=%v", current, err)
				}
			}
			ops := []readerbuilder.Operation{operation(calls[0], "shared"), operation(calls[1], "shared")}
			_, err = apply(version.SourceRevision+1, ops...)
			var conflict *sdk.Error
			if !errors.As(err, &conflict) || conflict.Code != sdk.ErrorConflict || conflict.Field != "sourceRevision" {
				t.Fatalf("stale revision accepted: %v", err)
			}
			unchanged()
			invalid := operation(calls[1], "shared")
			invalid.Function.ExpectedArgs = []string{"c", "'stale'"}
			rejected, err := apply(version.SourceRevision, ops[0], invalid)
			if err != nil || rejected.Applied {
				t.Fatalf("argument mismatch accepted: %+v err=%v", rejected, err)
			}
			unchanged()
			for _, missing := range []string{"missing", "policy-only", "disabled", "nil-entry"} {
				rejected, err = apply(version.SourceRevision, ops[0], operation(calls[1], missing))
				if err != nil || rejected.Applied {
					t.Fatalf("unavailable cache accepted: %+v err=%v", rejected, err)
				}
				found := false
				for _, d := range rejected.Inspection.Diagnostics {
					found = found || strings.Contains(d.Message, "no enabled configuration available to the reader builder service")
				}
				if !found {
					t.Fatalf("native validation missing: %+v", rejected)
				}
				unchanged()
			}
			saved, err := apply(version.SourceRevision, ops...)
			if err != nil || !saved.Applied {
				t.Fatalf("binding save failed: %+v err=%v", saved, err)
			}
			after := decode(saved.Inspection)
			want := before.Component.Clone()
			want.RootView.Source.Bindings.CacheName = "shared"
			want.RootView.Relations[0].View.Source.Bindings.CacheName = "shared"
			// Compare the public graph; native Clone also carries private metadata
			// that is intentionally absent from a decoded inspection response.
			wantJSON, _ := json.Marshal(want)
			afterJSON, _ := json.Marshal(after.Component)
			if !bytes.Equal(wantJSON, afterJSON) {
				t.Fatal("binding save changed unrelated storage, warmup, case, budget, or binding metadata")
			}
			native := readerbuilder.New(readerbuilder.Config{Name: before.Component.Name, Scope: before.Component.Key.Scope, AvailableCaches: before.AvailableCaches, AvailableConnectors: []string{"source"}}).Apply(f.ctx, readerbuilder.Request{DQL: source, Operation: readerbuilder.Operation{Type: readerbuilder.OperationBatch, Operations: ops}})
			if !native.Applied || saved.Inspection.DQL != native.DQL {
				t.Fatal("Studio rewrote source beyond the native guarded binding edits")
			}
			if !reflect.DeepEqual(after, decode(inspect())) {
				t.Fatal("binding metadata did not round-trip")
			}
			if _, err = f.db.ExecContext(f.ctx, "INSERT INTO component_acl(report_id,subject_type,subject_id,can_view,can_run,can_publish,can_use_dql) VALUES(?,'user','runner',1,1,1,0)", f.report.ID); err != nil {
				t.Fatal(err)
			}
			hidden, err := f.client.Versions().Inspect(sdk.WithPrincipal(f.ctx, sdk.Principal{Subject: "runner"}), f.report.ID, version.VersionNo)
			if err != nil {
				t.Fatal(err)
			}
			hiddenJSON, _ := json.Marshal(hidden)
			if hidden.Capabilities.CanUseDQL || strings.Contains(string(hiddenJSON), `"availableCaches"`) || strings.Contains(string(hiddenJSON), "PRIVATE_") {
				t.Fatalf("catalog/host settings leaked: %s", hiddenJSON)
			}
			// Native component-authored references remain accepted without catalog membership.
			f.transport.ReaderCaches = nil
			empty := inspect()
			var fields map[string]json.RawMessage
			if err = json.Unmarshal(empty.Structure, &fields); err != nil {
				t.Fatal(err)
			}
			if _, present := fields["availableCaches"]; present {
				t.Fatalf("empty catalog serialization changed: %s", empty.Structure)
			}
			current := decode(empty)
			var rootCall readerbuilder.FunctionOccurrence
			for _, call := range current.Functions {
				if call.Name == "use_cache" && call.Occurrence == 0 {
					rootCall = call
				}
			}
			fallback := "component"
			if namedComponent {
				fallback = "shared"
			}
			retained, err := apply(empty.Version.SourceRevision, operation(rootCall, fallback))
			if err != nil || !retained.Applied {
				t.Fatalf("native component reference rejected: %+v err=%v", retained, err)
			}
		})
	}
}
