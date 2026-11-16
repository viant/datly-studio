package preview

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/observability"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/spec"
)

func checkNativeExecutionLogging(t *testing.T, db *sql.DB, root string) {
	t.Helper()
	var original string
	if err := db.QueryRowContext(t.Context(), "SELECT authored_dql FROM component_versions WHERE report_id='vendors' AND version_no=1").Scan(&original); err != nil {
		t.Fatal(err)
	}
	authored := strings.Replace(original, "#define($_ = $Tenant", "#define($_ = $ID<*int>(query/id).Optional().WithPredicate(0,'equal','v','ID'))\n#define($_ = $Tenant", 1)
	authored = strings.Replace(authored, "$cache_warmup('ID', 'Tenant=1,2')", "$cache_warmup('ID', 'IndexParameter=ID')", 1)
	authored = strings.Replace(authored, "SELECT ID, NAME FROM VENDOR", `SELECT ID, NAME FROM VENDOR v ${predicate.Builder().CombineAnd($predicate.FilterGroup(0,"AND")).Build("WHERE")}`, 1)
	if _, err := db.ExecContext(t.Context(), "UPDATE component_versions SET authored_dql=?,generated_dql=? WHERE report_id='vendors' AND version_no=1", authored, authored); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := db.ExecContext(t.Context(), "UPDATE component_versions SET authored_dql=?,generated_dql=? WHERE report_id='vendors' AND version_no=1", original, original); err != nil {
			t.Error(err)
		}
	}()
	output, err := os.Create(filepath.Join(t.TempDir(), "native.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	enabled, includeSQL := true, false
	// Native Logging owns stdout independently of the summary logger.
	previous := os.Stdout
	os.Stdout = output
	owner, err := druntime.NewObservability(druntime.ObservabilityConfig{Logging: &observability.Logging{EnableAudit: &enabled, EnableTracing: &enabled, IncludeSQL: &includeSQL}})
	os.Stdout = previous
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Shutdown(t.Context())
	closed := 0
	dynamic := Dynamic{StudioDB: db, RootDir: root, Observability: owner, Dependencies: func(ctx context.Context, _ *spec.Component) (*DependencyRuntime, error) {
		return &DependencyRuntime{Options: []druntime.Option{druntime.WithResources(resource.New())}, Shutdown: func(ctx context.Context) error { closed++; return nil }}, nil
	}}
	position := 0
	logs := func() string {
		t.Helper()
		data, err := os.ReadFile(output.Name())
		if err != nil {
			t.Fatal(err)
		}
		result := string(data[position:])
		position = len(data)
		if strings.Contains(strings.ToUpper(result), "SELECT ") || strings.Contains(result, "FROM VENDOR") {
			t.Fatalf("IncludeSQL=false exposed SQL: %s", result)
		}
		return result
	}
	for i, want := range []string{"found_warmup=false found_lazy=false", "found_warmup=false found_lazy=true"} {
		if _, err := dynamic.Execute(t.Context(), "vendors", 1, sdk.PreviewInput{Input: json.RawMessage(`{"id":1}`)}); err != nil {
			t.Fatal(err)
		}
		if text := logs(); !strings.Contains(text, want) {
			t.Fatalf("preview %d missing cache evidence %q: %s", i, want, text)
		}
	}
	if _, err := dynamic.Warmup(t.Context(), "vendors", 1); err != nil {
		t.Fatal(err)
	}
	if text := logs(); !strings.Contains(text, "datly view read") || strings.Contains(text, "warmup started") || strings.Contains(text, "warmup completed") {
		t.Fatalf("unexpected native warmup events: %s", text)
	}
	if _, err := dynamic.Execute(t.Context(), "vendors", 1, sdk.PreviewInput{Input: json.RawMessage(`{"id":1}`)}); err != nil {
		t.Fatal(err)
	}
	if text := logs(); !strings.Contains(text, "found_warmup=true") {
		t.Fatalf("preview did not report a warmup hit: %s", text)
	}
	if _, err := dynamic.TestView(t.Context(), "vendors", 1, "vendors", sdk.ViewTestInput{}); err != nil {
		t.Fatal(err)
	}
	if text := logs(); text == "" {
		t.Fatal("selected-view execution emitted no native logs")
	}
	if closed != 5 {
		t.Fatalf("dependency runtimes closed=%d want=5", closed)
	}
	// Conflicting ownership is rejected by Datly, proving dependency options
	// survive composition. Cleanup must also run when runtime construction fails.
	other, err := druntime.NewObservability(druntime.ObservabilityConfig{})
	if err != nil {
		t.Fatal(err)
	}
	defer other.Shutdown(t.Context())
	dynamic.Dependencies = func(ctx context.Context, _ *spec.Component) (*DependencyRuntime, error) {
		return &DependencyRuntime{Options: []druntime.Option{druntime.WithManagedObservability(other)}, Shutdown: func(ctx context.Context) error { closed++; return nil }}, nil
	}
	if _, err := dynamic.Execute(t.Context(), "vendors", 1, sdk.PreviewInput{}); err == nil || !strings.Contains(err.Error(), "application owns observability configuration") {
		t.Fatalf("native dependency options were not preserved: %v", err)
	}
	if closed != 6 {
		t.Fatalf("failed runtime construction leaked dependencies: closed=%d", closed)
	}
	dynamic.Dependencies = nil
	if _, err := dynamic.Execute(t.Context(), "vendors", 1, sdk.PreviewInput{}); err != nil {
		t.Fatal(err)
	}
	if text := logs(); !strings.Contains(text, "datly view read") {
		t.Fatal("logging without a dependency resolver lost the native owner")
	}
}
