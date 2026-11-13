package preview

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/viant/datly/bootstrap"
	dexec "github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
)

type targetProofInput struct{}
type targetProofRow struct {
	ID    int `sqlx:"id"`
	Other int `sqlx:"other"`
}
type targetProofSummary struct {
	Total int `sqlx:"total"`
}
type targetProofOutput struct {
	Rows    []targetProofRow
	Summary *targetProofSummary
}

func TestNativeMaterializedWarmupTargets(t *testing.T) {
	directory := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, scenario := range []string{"named-root-and-child", "native-default-connectors", "plural-host-policies"} {
		rootPolicy := &spec.CacheWarmupSettings{Name: "root", IndexColumn: "id", Connector: "batch_root", IndexMeta: true}
		childPolicy := &spec.CacheWarmupSettings{Name: "child", IndexColumn: "total", Connector: "batch_child"}
		if scenario == "native-default-connectors" {
			rootPolicy.Connector = ""
			childPolicy.Connector = ""
		}
		profile := &spec.CacheSettings{Warmup: rootPolicy}
		if scenario == "plural-host-policies" {
			profile.Warmup = nil
			profile.Warmups = []*spec.CacheWarmupSettings{rootPolicy, {Name: "alternate", IndexColumn: "other", Connector: "batch_alternate"}}
		}
		child := &spec.View{Name: "summary", Source: &spec.ViewSource{SQL: "SELECT COUNT(*) AS total FROM ($View.NonWindowSQL) parent", Bindings: &spec.ViewBindings{Connector: "child_on_demand", CacheName: "shared", CacheWarmup: "child_profile"}}}
		root := &spec.View{Name: "records", Source: &spec.ViewSource{SQL: "SELECT 1 AS id, 2 AS other", Bindings: &spec.ViewBindings{Connector: "root_on_demand", CacheName: "shared", CacheWarmup: "root_profile"}}, Relations: []*spec.Relation{{Name: "summary", Holder: "Summary", Kind: spec.RelationKindDerived, Cardinality: spec.CardinalityOne, View: child}}}
		component := &spec.Component{Key: spec.Key{Kind: spec.KindComponent, Scope: "example.com/proof", Name: "Records"}, Routes: []*spec.Route{{Method: "GET", Path: "/records"}}, RootView: root}

		artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: reflect.TypeFor[targetProofInput](), OutputType: reflect.TypeFor[targetProofOutput](), DirectViewField: "Rows"})
		must(err)
		reader, err := artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: &dsql.SQLComponent{}, CacheSettings: map[string]*spec.CacheSettings{"shared": {Enabled: true, Provider: "afs", Location: filepath.Join(directory, scenario), TTL: "1m"}, "root_profile": profile, "child_profile": {Warmup: childPolicy}}})
		must(err)
		targeter, ok := reader.(dexec.ReaderWarmupTargeter)
		if !ok {
			t.Fatal("native reader does not expose targets")
		}
		targets := targeter.WarmupTargets()
		expected := 2
		if scenario == "plural-host-policies" {
			expected = 3
		}
		if len(targets) != expected {
			must(fmt.Errorf("%s: targets=%d want=%d", scenario, len(targets), expected))
		}
		var connectors []string
		for _, target := range targets {
			if target.Settings.Connector != "" {
				connectors = append(connectors, target.Settings.Connector)
			}
			t.Logf("%s view=%q policy=%q explicit_connector=%q index_meta=%v\n", scenario, target.View, target.Settings.Name, target.Settings.Connector, target.Settings.IndexMeta)
		}
		sort.Strings(connectors)
		want := []string{"batch_child", "batch_root"}
		if scenario == "native-default-connectors" {
			want = nil
		}
		if scenario == "plural-host-policies" {
			want = []string{"batch_alternate", "batch_child", "batch_root"}
		}
		if !reflect.DeepEqual(connectors, want) {
			t.Fatalf("%s connectors=%v want=%v", scenario, connectors, want)
		}

		if scenario == "native-default-connectors" && len(connectors) != 0 {
			t.Fatal("native default was replaced")
		}
		t.Logf("%s additional_connectors=%v; no SQL or warmup execution\n", scenario, connectors)
	}
}
