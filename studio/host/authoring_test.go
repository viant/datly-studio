package host_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/viant/datly-studio/studio/host"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	"github.com/viant/datly/transcribe"
)

func TestAuthoringSnapshotContextAndMutationIsolation(t *testing.T) {
	config := host.AuthoringConfig{ModulePath: "example.com/owned", Config: host.Config{PredicatePackages: []predicatecatalog.Package{{Alias: "tenant", Path: "example.com/acme/iam", Types: []reflect.Type{reflect.TypeFor[tenantPredicate]()}, AuthorizationTypes: []string{"tenantPredicate"}}}}}
	snapshot, err := host.NewAuthoring(config)
	if err != nil {
		t.Fatal(err)
	}
	config.ModulePath = "evil.example/changed"
	config.Config.PredicatePackages[0].Path = "evil.example/changed"
	config.Config.PredicatePackages[0].Types[0] = nil
	config.Config.PredicatePackages[0].AuthorizationTypes[0] = "changed"
	snapshot.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current, err := host.AuthoringFromContext(r.Context())
		if err != nil {
			t.Fatal(err)
		}
		if current.ModulePath() != "example.com/owned" || !current.PredicateCatalog().Contains("example.com/acme/iam", "tenantPredicate") {
			t.Fatal("caller changed captured authority")
		}
	})).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	legacy, err := host.AuthoringFromContext(context.Background())
	if err != nil || legacy.ModulePath() != host.DefaultAuthoringModule {
		t.Fatalf("legacy=%v err=%v", legacy, err)
	}
	for _, invalid := range []string{"../escape", "example.com/../escape", "https://example.com/owned", "example.com/owned/"} {
		if _, err := host.NewAuthoring(host.AuthoringConfig{ModulePath: invalid}); err == nil {
			t.Fatalf("accepted %q", invalid)
		}
	}
}

func TestAuthoringTrustedModuleRuntimeContract(t *testing.T) {
	// Same authored package/source is compiled under separate trusted host snapshots.
	source := &transcribe.Source{Scope: "example.com/owned/read", Name: "reader", Text: `#package('example.com/owned/read')
#setting($_ = $route('/records','GET'))
SELECT 1 AS id`}
	for _, test := range []struct {
		module  string
		allowed bool
	}{{"", false}, {"example.com/owned", true}, {"example.com/other", false}} {
		snapshot, err := host.NewAuthoring(host.AuthoringConfig{ModulePath: test.module})
		if err != nil {
			t.Fatal(err)
		}
		source.Types, err = snapshot.RuntimeTypes()
		if err != nil {
			t.Fatal(err)
		}
		_, err = transcribe.NewCompiler().RuntimeContractsInModule(context.Background(), snapshot.ModulePath(), source)
		if (err == nil) != test.allowed {
			t.Fatalf("module %q allowed=%v error=%v", test.module, test.allowed, err)
		}
	}
}
