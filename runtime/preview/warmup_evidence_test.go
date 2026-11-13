package preview

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/viant/datly/bootstrap"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/sqlx/io/read/cache"
	"github.com/viant/sqlx/io/read/cache/aerospike"
)

type opaqueEvidenceCache struct{ cache.Cache }

func TestWarmupEvidenceResolvedNativeMetadata(t *testing.T) {
	for _, tc := range []struct {
		name, logical, backend string
		supplied               cache.Cache
	}{
		{name: "named default AFS", logical: "shared", backend: "afs"},
		{name: "named Aerospike", logical: "aerospike", backend: "aerospike", supplied: &aerospike.Cache{}},
		{name: "opaque service", logical: "aerospike", supplied: &opaqueEvidenceCache{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, expanded := range []bool{false, true} {
				policy := &spec.CacheWarmupSettings{Name: "root-policy", IndexColumn: "id", Connector: "batch", IndexMeta: expanded}
				root := &spec.View{Name: "records", Source: &spec.ViewSource{SQL: "SELECT 1 AS id, 2 AS other", Bindings: &spec.ViewBindings{CacheName: tc.logical}}}
				if expanded {
					root.Relations = []*spec.Relation{{Name: "summary", Holder: "Summary", Kind: spec.RelationKindDerived, Cardinality: spec.CardinalityOne, View: &spec.View{Name: "summary", Source: &spec.ViewSource{SQL: "SELECT COUNT(*) AS total FROM ($View.NonWindowSQL) parent", Bindings: &spec.ViewBindings{CacheName: "summary-cache", CacheWarmup: "child-policy"}}}}}
				}
				component := &spec.Component{Key: spec.Key{Kind: spec.KindComponent, Scope: "example.com/evidence", Name: "Records"}, Routes: []*spec.Route{{Method: "GET", Path: "/records"}}, RootView: root, Settings: &spec.Settings{Cache: &spec.CacheSettings{Enabled: true, Name: tc.logical, Warmup: policy}}}
				artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: reflect.TypeFor[targetProofInput](), OutputType: reflect.TypeFor[targetProofOutput](), DirectViewField: "Rows"})
				if err != nil {
					t.Fatal(err)
				}
				storage := &spec.CacheSettings{Enabled: true, Location: filepath.Join(t.TempDir(), "private-storage"), TTL: "1m"}
				cfg := bootstrap.ReaderRuntimeConfig{SQL: &dsql.SQLComponent{}, CacheSettings: map[string]*spec.CacheSettings{tc.logical: storage, "summary-cache": {Enabled: true, Location: t.TempDir(), TTL: "1m"}, "child-policy": {Warmup: &spec.CacheWarmupSettings{Name: "child-policy", IndexColumn: "total"}}}}
				if tc.supplied != nil {
					// A supplied service is authoritative even if host configuration
					// contains an address. No network or cache execution occurs here.
					storage.Provider = "aerospike://PRIVATE_CREDENTIAL@unreachable.invalid:3000/PRIVATE_STORAGE"
					cfg.ReadCaches = map[string]cache.Cache{"records": tc.supplied}
				}
				reader, err := artifact.ReaderCompilation().NewExecution(cfg)
				if err != nil {
					t.Fatal(err)
				}
				targets := reader.(dexec.ReaderWarmupTargeter).WarmupTargets()
				evidence := warmupTargetEvidence("records", targets)
				if len(evidence.ScheduledTargets) != len(targets) {
					t.Fatalf("scheduled targets changed: %+v", evidence)
				}
				first := evidence.ScheduledTargets[0]
				if first.View != "" || first.PolicyName != "root-policy" || first.CacheName != tc.logical || first.CacheProvider != tc.backend || first.ConnectorName != "batch" || first.IndexColumn != "id" || first.IndexMeta != expanded {
					t.Fatalf("native policy lost: %+v", first)
				}
				if first.Work[0].View != "records" || first.Work[0].CacheName != tc.logical || first.Work[0].CacheProvider != tc.backend {
					t.Fatalf("root work=%+v", first.Work)
				}
				if expanded {
					if len(targets) != 2 || len(first.Work) != 2 || first.Work[1].View != "summary" || first.Work[1].CacheProvider != "afs" || first.Work[1].CacheName != "summary-cache" || evidence.ScheduledTargets[1].PolicyName != "child-policy" {
						t.Fatalf("expanded work=%+v", evidence)
					}
					if evidence.CacheName != "" || evidence.CacheProvider != "" {
						t.Fatalf("legacy cache implies one backend: %+v", evidence)
					}
				} else if evidence.CacheName != tc.logical || evidence.CacheProvider != tc.backend || len(first.Work) != 1 {
					t.Fatalf("single-target compatibility=%+v", evidence)
				}
				raw, err := json.Marshal(evidence)
				if err != nil {
					t.Fatal(err)
				}
				for _, secret := range []string{"PRIVATE_", "unreachable.invalid", storage.Location, "settings", "cases"} {
					if strings.Contains(string(raw), secret) {
						t.Fatalf("storage or settings leaked: %s", raw)
					}
				}
				// Native snapshots and the SDK projection own independent values.
				targets[0].Settings.Name = "changed"
				targets[0].Work[0].Settings.Name = "changed"
				if evidence.ScheduledTargets[0].PolicyName != "root-policy" {
					t.Fatal("evidence retained mutable settings")
				}
			}
		})
	}
}

type evidenceCountingReader struct {
	dexec.Reader
	dexec.ReaderWarmupTargeter
	dexec.QueryPreparer
	warmer dexec.ReaderWarmer
	calls  int
}

func (r *evidenceCountingReader) Warmup(ctx context.Context, invocation dexec.ReaderWarmupInvocation) (int, error) {
	r.calls++
	return r.warmer.Warmup(ctx, invocation)
}

func TestWarmupEvidenceDoesNotScheduleIndexMetaWork(t *testing.T) {
	ctx := t.Context()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "source.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.ExecContext(ctx, "CREATE TABLE RECORDS(ID INTEGER, OTHER INTEGER); INSERT INTO RECORDS VALUES(1,2),(2,3)"); err != nil {
		t.Fatal(err)
	}
	root := &spec.View{Name: "records", Source: &spec.ViewSource{SQL: "SELECT ID AS id, OTHER AS other FROM RECORDS", Bindings: &spec.ViewBindings{Connector: "main", CacheName: "root"}}, Relations: []*spec.Relation{{Name: "summary", Holder: "Summary", Kind: spec.RelationKindDerived, Cardinality: spec.CardinalityOne, View: &spec.View{Name: "summary", Source: &spec.ViewSource{SQL: "SELECT COUNT(*) AS total FROM ($View.NonWindowSQL) parent", Bindings: &spec.ViewBindings{CacheName: "summary"}}}}}}
	component := &spec.Component{Key: spec.Key{Kind: spec.KindComponent, Scope: "example.com/evidence", Name: "Records"}, Settings: &spec.Settings{DefaultConnector: "main", Cache: &spec.CacheSettings{Enabled: true, Name: "root", Warmup: &spec.CacheWarmupSettings{Name: "root-policy", IndexColumn: "id", IndexMeta: true}}}, Routes: []*spec.Route{{Method: "GET", Path: "/records"}}, RootView: root}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: reflect.TypeFor[targetProofInput](), OutputType: reflect.TypeFor[targetProofOutput](), DirectViewField: "Rows"})
	if err != nil {
		t.Fatal(err)
	}
	source := &dsql.SQLComponent{DB: db}
	if err = source.RegisterConnector("main", db); err != nil {
		t.Fatal(err)
	}
	cfg := bootstrap.ReaderRuntimeConfig{SQL: source, CacheSettings: map[string]*spec.CacheSettings{"root": {Enabled: true, Location: filepath.Join(t.TempDir(), "root"), TTL: "1m"}, "summary": {Enabled: true, Location: filepath.Join(t.TempDir(), "summary"), TTL: "1m"}}}
	reader, err := artifact.ReaderCompilation().NewExecution(cfg)
	if err != nil {
		t.Fatal(err)
	}
	counted := &evidenceCountingReader{Reader: reader, ReaderWarmupTargeter: reader.(dexec.ReaderWarmupTargeter), QueryPreparer: reader.(dexec.QueryPreparer), warmer: reader.(dexec.ReaderWarmer)}
	providers, err := artifact.ReaderCompilation().NewProviders(cfg)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := artifact.Registration(druntime.RegisteredComponent{Reader: counted, Providers: providers})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := druntime.NewRuntime([]*druntime.RegisteredComponent{entry})
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Shutdown(ctx)
	operation, err := runtime.NewWarmup(dexec.ComponentTarget{Component: component.Key, Route: spec.RouteRef{Method: "GET", Path: "/records"}})
	if err != nil {
		t.Fatal(err)
	}
	planned, err := operation.PlannedCases()
	if err != nil {
		t.Fatal(err)
	}
	operation, err = operation.Prepare(ctx)
	if err != nil {
		t.Fatal(err)
	}
	evidence := warmupTargetEvidence(root.Name, counted.WarmupTargets())
	if len(evidence.ScheduledTargets) != 1 || len(evidence.ScheduledTargets[0].Work) != 2 {
		t.Fatalf("IndexMeta evidence=%+v", evidence)
	}
	entries, err := operation.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if planned != 1 || counted.calls != 1 || entries < 1 {
		t.Fatalf("work changed execution: planned=%d calls=%d entries=%d", planned, counted.calls, entries)
	}
}
