package preview

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/viant/bindly/provider/values"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/bootstrap"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/typecatalog"
	"github.com/viant/x"
	xhandler "github.com/viant/xdatly/handler"
	handlerexec "github.com/viant/xdatly/handler/exec"
	xpredicate "github.com/viant/xdatly/predicate"
)

type PreviewIdentity struct{ ID int }
type PreviewOwnerPredicate struct{}

type warmupPredicateCallsKey struct{}
type warmupInvocationPhasesKey struct{}

func (*PreviewOwnerPredicate) Compute(ctx context.Context, value any) (*xpredicate.Criteria, error) {
	if calls, ok := ctx.Value(warmupPredicateCallsKey{}).(*int); ok {
		*calls++
	}
	if phases, ok := ctx.Value(warmupInvocationPhasesKey{}).(*[]handlerexec.WarmupPhase); ok {
		invocation := handlerexec.InvocationFromContext(ctx)
		if invocation == nil || !invocation.MayBypassRowAuthorization() {
			return nil, fmt.Errorf("native warmup invocation unavailable")
		}
		*phases = append(*phases, invocation.WarmupPhase())
	}
	identity, ok := value.(*PreviewIdentity)
	if !ok || identity == nil {
		return &xpredicate.Criteria{Expression: "1=0"}, nil
	}
	return &xpredicate.Criteria{Expression: "ID = ?", Placeholders: []any{identity.ID}}, nil
}

type previewIdentityInput struct {
	Token string `parameter:"Token,kind=header,in=Authorization,required=true"`
}
type previewIdentityHandler struct{}
type warmupIdentityCallsKey struct{}

func (previewIdentityHandler) Exec(ctx context.Context, _ xhandler.Session, input *previewIdentityInput, output *PreviewIdentity) error {
	if calls, ok := ctx.Value(warmupIdentityCallsKey{}).(*int); ok {
		*calls++
	}
	switch input.Token {
	case "Bearer alice":
		output.ID = 1
	case "Bearer bob":
		output.ID = 2
	default:
		return fmt.Errorf("identity rejected")
	}
	return nil
}

func checkPreviewDependencyIdentity(t *testing.T, db *sql.DB, root, original string) {
	// The fixture explicitly opts this namespace into its linked test handler.
	if _, err := db.ExecContext(context.Background(), `INSERT OR IGNORE INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at) VALUES('owner','general','General','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP); INSERT OR REPLACE INTO namespace_settings(owner_id,namespace,setting_key,value_json,updated_at,updated_by) VALUES('owner','general','security','{"predicatePackages":["github.com/viant/datly-studio/runtime/preview"]}',CURRENT_TIMESTAMP,'fixture')`); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	dql := `#package('example.com/app/dynamic/vendors')
#import('identity','github.com/viant/datly-studio/runtime/preview')
#import('time','time')
#setting($_ = $connector('vendor'))
#setting($_ = $route('/vendors','GET'))
#define($_ = $Identity<*identity.PreviewIdentity>(component/GET:/identity).WithPredicate(0,'handler','identity.PreviewOwnerPredicate'))
#define($_ = $From<*time.Time>(form/from).Tag('format:"dateFormat=YYYY-MM-DD"').Optional().WithPredicate(0,'expr','? IS NOT NULL'))
#define($_ = $Vendors<[]*Vendor>(output/view))
SELECT vendors.*,type(vendors,'Vendor') FROM (SELECT ID,NAME FROM VENDOR ${predicate.Builder().CombineAnd($predicate.FilterGroup(0,"AND")).Build("WHERE")}) vendors`
	if _, err := db.ExecContext(ctx, "UPDATE component_versions SET authored_dql=?,generated_dql=? WHERE report_id='vendors' AND version_no=1", dql, dql); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := db.ExecContext(ctx, "UPDATE component_versions SET authored_dql=?,generated_dql=? WHERE report_id='vendors' AND version_no=1", original, original); err != nil {
			t.Fatal(err)
		}
	}()
	types := typecatalog.NewCatalog()
	for _, typ := range []reflect.Type{reflect.TypeFor[PreviewIdentity](), reflect.TypeFor[PreviewOwnerPredicate]()} {
		if err := types.Register(typecatalog.TypeOriginPackage, x.NewType(typ)); err != nil {
			t.Fatal(err)
		}
	}
	closed := 0
	dynamic := Dynamic{StudioDB: db, RootDir: root, Types: types, Dependencies: func(context.Context, *spec.Component) (*DependencyRuntime, error) {
		artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: &spec.Component{Key: spec.Key{Kind: spec.KindComponent, Scope: "test/identity", Name: "Identity"}, Routes: []*spec.Route{{Method: "GET", Path: "/identity", Internal: true}}}, InputType: reflect.TypeFor[previewIdentityInput](), OutputType: reflect.TypeFor[PreviewIdentity]()})
		if err != nil {
			return nil, err
		}
		registered, err := artifact.Registration(registry.RegisteredComponent{Handler: custom.New[previewIdentityInput, PreviewIdentity](previewIdentityHandler{})})
		if err != nil {
			return nil, err
		}
		return &DependencyRuntime{Components: []*registry.RegisteredComponent{registered}, Shutdown: func(context.Context) error { closed++; return nil }}, nil
	}}
	for _, test := range []struct{ token, want, absent string }{{"alice", "Northwind", "Contoso"}, {"bob", "Contoso", "Northwind"}} {
		verified := sdk.WithVerifiedCredential(sdk.WithPrincipal(ctx, sdk.Principal{Subject: test.token}), sdk.VerifiedCredential{Bearer: test.token, Claims: struct{}{}})
		result, err := dynamic.Execute(verified, "vendors", 1, sdk.PreviewInput{Input: []byte(`{"from":"2026-09-01"}`), Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(result.Data), test.want) || strings.Contains(string(result.Data), test.absent) {
			t.Fatalf("predicate result for %s: %s", test.token, result.Data)
		}
		if _, err = dynamic.Execute(verified, "vendors", 1, sdk.PreviewInput{Input: []byte(`{"Identity":{"ID":3}}`)}); err == nil {
			t.Fatal("client replaced identity dependency")
		}
	}
	if _, err := dynamic.Execute(ctx, "vendors", 1, sdk.PreviewInput{}); err == nil {
		t.Fatal("missing authentication accepted")
	}
	if closed != 5 {
		t.Fatalf("dependency cleanup count=%d", closed)
	}
	checkWarmupDependencies(ctx, t, db, root, dql, types)
	checkWarmupJWTCredentials(ctx, t, db, root, dql, types)
}

// This fixture consumes the server-verified context directly. The separate
// header-bound fixtures below distinguish plain headers from native JWT inputs.
type warmupContextIdentityInput struct{}
type warmupContextIdentityHandler struct{}

func (warmupContextIdentityHandler) Exec(ctx context.Context, _ xhandler.Session, _ *warmupContextIdentityInput, output *PreviewIdentity) error {
	credential, ok := sdk.VerifiedCredentialFromContext(ctx)
	if !ok || credential.Bearer != "alice" {
		return fmt.Errorf("identity rejected")
	}
	output.ID = 1
	return nil
}

func checkWarmupDependencies(ctx context.Context, t *testing.T, db *sql.DB, root, source string, types *typecatalog.Catalog) {
	t.Helper()
	dql := strings.Replace(source, "SELECT vendors.*", fmt.Sprintf("#setting($_ = $cache('vendors', '1m').WithLocation('%s'))\n#setting($_ = $cache_warmup('ID', 'Limit=0'))\nSELECT vendors.*", filepath.Join(t.TempDir(), "cache")), 1)
	if _, err := db.ExecContext(ctx, "UPDATE component_versions SET authored_dql=?,generated_dql=? WHERE report_id='vendors' AND version_no=1", dql, dql); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := db.ExecContext(ctx, "UPDATE component_versions SET authored_dql=?,generated_dql=? WHERE report_id='vendors' AND version_no=1", source, source); err != nil {
			t.Error(err)
		}
	}()
	for _, tc := range []struct {
		name, token                                string
		header, missing, duplicate, headerProvider bool
		want                                       string
	}{
		{name: "resolved dependency and authorization", token: "alice"},
		{name: "missing credential", want: "identity rejected"},
		{name: "invalid credential", token: "invalid", want: "identity rejected"},
		{name: "non-JWT headers are not inferred", token: "alice", header: true, want: "Bad Request"},
		{name: "native header provider and runtime options", token: "alice", header: true, headerProvider: true},
		{name: "unresolved dependency", token: "alice", missing: true, want: "component dependency route contract not found"},
		{name: "runtime initialization cleanup", token: "alice", duplicate: true, want: "initialize dynamic warmup runtime"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			closed, resolved, predicates, identityCalls := 0, 0, 0, 0
			callCtx := context.WithValue(ctx, warmupPredicateCallsKey{}, &predicates)
			callCtx = context.WithValue(callCtx, warmupIdentityCallsKey{}, &identityCalls)
			if tc.token != "" {
				callCtx = sdk.WithVerifiedCredential(callCtx, sdk.VerifiedCredential{Bearer: tc.token, Claims: struct{}{}})
			}
			dynamic := Dynamic{StudioDB: db, RootDir: root, Types: types}
			if !tc.missing {
				dynamic.Dependencies = func(ctx context.Context, _ *spec.Component) (*DependencyRuntime, error) {
					resolved++
					inputType := reflect.TypeFor[warmupContextIdentityInput]()
					handler := custom.New[warmupContextIdentityInput, PreviewIdentity](warmupContextIdentityHandler{})
					if tc.header {
						inputType = reflect.TypeFor[previewIdentityInput]()
						handler = custom.New[previewIdentityInput, PreviewIdentity](previewIdentityHandler{})
					}
					artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: &spec.Component{Key: spec.Key{Kind: spec.KindComponent, Scope: "test/identity", Name: "Identity"}, Routes: []*spec.Route{{Method: "GET", Path: "/identity", Internal: true}}}, InputType: inputType, OutputType: reflect.TypeFor[PreviewIdentity]()})
					if err != nil {
						return nil, err
					}
					entry, err := artifact.Registration(registry.RegisteredComponent{Handler: handler})
					if err != nil {
						return nil, err
					}
					entries := []*registry.RegisteredComponent{entry}
					if tc.duplicate {
						entries = append(entries, entry)
					}
					var options []druntime.Option
					if tc.headerProvider {
						credential, _ := sdk.VerifiedCredentialFromContext(ctx)
						options = append(options, druntime.WithApplicationProviders(values.New("header", map[string]any{"Authorization": "Bearer " + credential.Bearer})))
					}
					return &DependencyRuntime{Components: entries, Options: options, Shutdown: func(context.Context) error { closed++; return nil }}, nil
				}
			}
			result, err := dynamic.Warmup(callCtx, "vendors", 1)
			if tc.want != "" {
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("warmup error=%v, want %q", err, tc.want)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if result.Entries < 1 || result.CompletedCases < 1 || predicates < 1 {
					t.Fatalf("warmup=%+v predicate calls=%d", result, predicates)
				}
			}
			if tc.header && !tc.headerProvider && identityCalls != 0 {
				t.Fatalf("required header did not stop dependency execution: calls=%d", identityCalls)
			}
			if !tc.missing && (resolved != 1 || closed != 1) {
				t.Fatalf("resolved=%d cleanup=%d", resolved, closed)
			}
		})
	}
}
